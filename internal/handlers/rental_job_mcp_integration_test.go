package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/schema"
)

func TestRentalJobOwnerAtomicFieldsAndLifecycle(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("dedicated _test database required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	const ns = "rental_job_owner_test"
	exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE;CREATE SCHEMA ` + ns + `;SET search_path TO ` + ns)
	defer db.Exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE`)
	fixture, err := os.ReadFile("testdata/rental_job_mcp.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(fixture))
	if err = schema.EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("job-test-", 6))
	gin.SetMode(gin.TestMode)
	h := &RentalJobMCP{db: db}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1, IsAdmin: true, IsActive: true}) })
	r.POST("/jobs/:operation", h.Change)
	invoke := func(op string, in map[string]any, key string, financial bool) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(in)
		req := httptest.NewRequest("POST", "/jobs/"+op, bytes.NewReader(raw))
		req.Header.Set("X-Cores-Origin", "MCP/AI")
		req.Header.Set("Idempotency-Key", key)
		action := op
		if op == "restore" {
			action = "archive"
		}
		token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": "cores:rental:" + action, "mcp_financial": financial, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		if e != nil {
			t.Fatal(e)
		}
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		out := map[string]any{}
		if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	prepare := func(op string, in map[string]any, financial bool) (map[string]any, map[string]any) {
		t.Helper()
		copy := map[string]any{}
		for k, v := range in {
			copy[k] = v
		}
		copy["preview"] = true
		s, p := invoke(op, copy, "", financial)
		if s != 200 {
			t.Fatalf("preview %d %#v", s, p)
		}
		delete(copy, "preview")
		if p["expected_updated_at"] != nil {
			copy["expected_updated_at"] = p["expected_updated_at"]
		}
		copy["expected_context"] = p["expected_context"]
		copy["confirmation_text"] = p["required_confirmation_text"]
		copy["confirm_change"] = true
		return copy, p
	}
	execute := func(op string, in map[string]any, key string, financial bool) map[string]any {
		t.Helper()
		s, out := invoke(op, in, key, financial)
		if s < 200 || s > 299 || out["job"] == nil {
			t.Fatalf("execute %d %#v", s, out)
		}
		return out
	}
	required := func(p map[string]any, field string) {
		t.Helper()
		raw, _ := json.Marshal(p["required_fields"])
		if !bytes.Contains(raw, []byte(`"`+field+`"`)) {
			t.Fatalf("missing guard %s %#v", field, p)
		}
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	_, p := prepare("create", map[string]any{}, false)
	required(p, "customer_id")
	required(p, "description")
	if s, _ := invoke("create", map[string]any{"description": "No commercial scope", "customer_id": 1, "revenue": 12}, "", false); s != 403 {
		t.Fatal("commercial scope bypass", s)
	}
	in, p := prepare("create", map[string]any{"description": "Complete MCP job", "customer_id": 1, "status_id": 2, "job_category_id": 1, "venue_id": 1, "start_date": "2030-10-01", "end_date": "2030-10-03", "revenue": 200, "discount": 10, "discount_type": "percent", "multiply_by_days": false, "prices_include_tax": true}, true)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	created := execute("create", in, "job-create-all-fields", true)
	job := created["job"].(map[string]any)
	id := int(job["job_id"].(float64))
	if job["final_revenue"] != float64(180) || job["multiply_by_days"] != false || job["prices_include_tax"] != true || job["created_by"] != float64(1) || job["job_code"] != fmt.Sprintf("JOB%06d", id) {
		t.Fatal(job)
	}
	s, replay := invoke("create", in, "job-create-all-fields", true)
	if s != 201 || !reflect.DeepEqual(created, replay) || count("jobs") != 1 || count("audit_log") != 1 || count("job_history") != 1 {
		t.Fatal("durable create replay", s, replay)
	}
	exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ = invoke("create", in, "job-create-all-fields", true); s != 403 {
		t.Fatal("cached actor rights bypass", s)
	}
	exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	_, p = prepare("create", map[string]any{"description": "Complete MCP job", "customer_id": 1, "start_date": "2030-10-01", "end_date": "2030-10-03"}, false)
	required(p, "review_duplicate_and_allow_explicitly")
	in, p = prepare("update", map[string]any{"job_id": id, "description": "Changed title", "discount": 20, "clear_fields": []string{"job_category_id", "venue_id"}}, true)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	updated := execute("update", in, "job-update-complete", true)
	job = updated["job"].(map[string]any)
	if job["venue_id"] != nil || job["job_category_id"] != nil || job["final_revenue"] != float64(160) {
		t.Fatal(job)
	}
	stale, _ := prepare("update", map[string]any{"job_id": id, "description": "Should become stale"}, false)
	exec(`INSERT INTO job_devices(jobid,deviceid) VALUES($1,'JOB-TEST-DEVICE')`, id)
	s, p = invoke("update", stale, "job-stale-child", false)
	if s != 200 || p["operation_status"] != "needs_input" {
		t.Fatal("child change bypass", s, p)
	}
	required(p, "expected_updated_at")
	required(p, "expected_context")
	in, _ = prepare("update", map[string]any{"job_id": id, "description": "Device-bound preview"}, false)
	exec(`UPDATE devices SET updated_at=clock_timestamp(),condition_status='damaged' WHERE deviceid='JOB-TEST-DEVICE'`)
	_, p = invoke("update", in, "job-stale-device", false)
	required(p, "expected_context")
	exec(`INSERT INTO job_edit_sessions(job_id,user_id) VALUES($1,2)`, id)
	_, p = prepare("update", map[string]any{"job_id": id, "description": "Blocked editor"}, false)
	required(p, "active_editing_session")
	exec(`DELETE FROM job_edit_sessions`)
	exec(`UPDATE job_devices SET pack_status='issued' WHERE jobid=$1`, id)
	_, p = prepare("archive", map[string]any{"job_id": id}, false)
	required(p, "active_warehouse_dependencies")
	exec(`UPDATE job_devices SET pack_status='returned' WHERE jobid=$1`, id)
	exec(`INSERT INTO warehouse_tasks(job_id,status) VALUES($1,'open')`, id)
	_, p = prepare("archive", map[string]any{"job_id": id}, false)
	required(p, "active_warehouse_dependencies")
	exec(`UPDATE warehouse_tasks SET status='done'`)
	exec(`INSERT INTO cases(current_job_id) VALUES($1)`, id)
	_, p = prepare("archive", map[string]any{"job_id": id}, false)
	required(p, "active_warehouse_dependencies")
	exec(`UPDATE cases SET current_job_id=NULL`)
	in, p = prepare("archive", map[string]any{"job_id": id}, false)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	archived := execute("archive", in, "job-archive", false)
	if archived["job"].(map[string]any)["status_id"] != float64(6) {
		t.Fatal(archived)
	}
	_, p = prepare("update", map[string]any{"job_id": id, "description": "Archived edit"}, false)
	required(p, "lifecycle_state")
	if _, e := db.Exec(`DELETE FROM job_devices WHERE jobid=$1`, id); e == nil {
		t.Fatal("archived contents removed")
	}
	if _, e := db.Exec(`DELETE FROM jobs WHERE jobid=$1`, id); e == nil {
		t.Fatal("job history deleted")
	}
	in, p = prepare("restore", map[string]any{"job_id": id}, false)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	restored := execute("restore", in, "job-restore", false)
	if restored["job"].(map[string]any)["status_id"] != float64(6) || count("job_devices") != 1 {
		t.Fatal("restore changed historical job", restored)
	}
	in, p = prepare("update", map[string]any{"job_id": id, "status_id": 1, "clear_fields": []string{"start_date", "end_date"}}, false)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	execute("update", in, "job-reopen", false)
	exec(`INSERT INTO job_positions(job_id,quantity,unit_price,follow_day_factor,tax_rate) VALUES($1,3,100,0.5,19)`, id)
	_, p = prepare("update", map[string]any{"job_id": id, "start_date": "2030-10-01", "end_date": "2030-10-04"}, false)
	required(p, "financial_effect_scope")
	in, p = prepare("update", map[string]any{"job_id": id, "start_date": "2030-10-01", "end_date": "2030-10-04", "multiply_by_days": true, "prices_include_tax": false}, true)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	calculated := execute("update", in, "job-recalculate-positions", true)
	if calculated["job"].(map[string]any)["revenue"] != float64(714) || calculated["job"].(map[string]any)["final_revenue"] != float64(571.2) {
		t.Fatal(calculated)
	}
	_, p = prepare("update", map[string]any{"job_id": id, "revenue": 1}, true)
	required(p, "revenue_owned_by_positions")
	in, p = prepare("update", map[string]any{"job_id": id, "description": "Atomic audit retry"}, true)
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	n := count("rental_mcp_mutation_receipts")
	hist := count("job_history")
	exec(`CREATE FUNCTION fail_job_audit() RETURNS TRIGGER LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'test audit failure';END;$$;CREATE TRIGGER fail_job_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_job_audit()`)
	s, p = invoke("update", in, "job-audit-retry", true)
	if s < 400 || count("rental_mcp_mutation_receipts") != n || count("job_history") != hist {
		t.Fatal("non-atomic failed audit", s, p)
	}
	exec(`DROP TRIGGER fail_job_audit ON audit_log`)
	done := execute("update", in, "job-audit-retry", true)
	// Durable replay survives handler replacement and later native changes.
	h = &RentalJobMCP{db: db}
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	r.POST("/jobs/:operation", h.Change)
	exec(`UPDATE jobs SET description='Later native change' WHERE jobid=$1`, id)
	s, replay = invoke("update", in, "job-audit-retry", true)
	if s != 200 || !reflect.DeepEqual(done, replay) {
		t.Fatal("restart replay", s, replay)
	}
	for _, bad := range []map[string]any{{"job_id": id, "delete": true}, {"job_id": id, "final_revenue": 1}, {"job_id": id, "m365_event_id": "injected"}, {"job_id": id, "clear_fields": []string{"customer_id"}}} {
		if s, _ := invoke("update", bad, "", true); s != 400 {
			t.Fatal("injected field", s, bad)
		}
	}
}
