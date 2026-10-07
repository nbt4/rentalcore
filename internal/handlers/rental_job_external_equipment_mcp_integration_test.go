package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
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

func TestRentalJobExternalEquipmentAtomicAssignmentAndGuards(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("dedicated _test DB required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DROP SCHEMA IF EXISTS rental_external_equipment_owner_test CASCADE;CREATE SCHEMA rental_external_equipment_owner_test;SET search_path TO rental_external_equipment_owner_test`)
	defer db.Exec(`DROP SCHEMA IF EXISTS rental_external_equipment_owner_test CASCADE`)
	fixture, err := os.ReadFile("testdata/rental_job_mcp.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(fixture))
	// Use the existing assignment migration, not a model with legacy column names.
	exec(`DROP TABLE job_rental_equipment;CREATE TABLE rental_equipment(id SERIAL PRIMARY KEY,name TEXT,supplier TEXT,category TEXT,rental_price NUMERIC(12,2),customer_price NUMERIC(12,2),is_active BOOL DEFAULT true,updated_at TIMESTAMP DEFAULT NOW());
 INSERT INTO rental_equipment(id,name,supplier,rental_price,customer_price) VALUES(1,'Test rental one','Test supplier',12.35,20),(2,'Test rental two','Test supplier',7.50,15),(3,'Missing catalog price','Test supplier',NULL,NULL),(4,'Test free rental','Test supplier',0,0),(5,'Test costly rental','Test supplier',9999999999.99,0);
 INSERT INTO jobs(jobid,job_code,customerid,statusid,description,startdate,enddate,revenue,final_revenue) VALUES(1,'JOB_TEST_RENTAL',1,1,'Rental assignment fixture','2026-10-01','2026-10-04',500,450)`)
	assignmentMigration, err := os.ReadFile("../../migrations/039_job_rental_equipment.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(assignmentMigration))
	if err = schema.EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("external-rental-test-", 3))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	actor := uint(1)
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: actor}) })
	h := &RentalJobMCP{db: db}
	router.POST("/api/v1/mcp/jobs/:operation", h.Change)
	financial, action, origin := true, "create", "MCP/AI"
	invokeRaw := func(raw []byte, key string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/v1/mcp/jobs/external-equipment-create", bytes.NewReader(raw))
		req.Header.Set("X-Cores-Origin", origin)
		req.Header.Set("Idempotency-Key", key)
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": "cores:rental:" + action, "mcp_financial": financial, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		out := map[string]any{}
		if err = json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	invoke := func(in map[string]any, key string) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(in)
		return invokeRaw(raw, key)
	}
	prepare := func(in map[string]any) (map[string]any, map[string]any) {
		t.Helper()
		copy := map[string]any{}
		for k, v := range in {
			copy[k] = v
		}
		copy["preview"] = true
		status, p := invoke(copy, "")
		if status != 200 {
			t.Fatal("preview", status, p)
		}
		delete(copy, "preview")
		copy["expected_job_updated_at"] = p["expected_job_updated_at"]
		copy["expected_context"] = p["expected_context"]
		copy["confirmation_text"] = p["required_confirmation_text"]
		copy["confirm_change"] = true
		return copy, p
	}
	required := func(p map[string]any, key string) {
		t.Helper()
		raw, _ := json.Marshal(p["required_fields"])
		if !bytes.Contains(raw, []byte(`"`+key+`"`)) {
			t.Fatal("missing guard", key, p)
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
	base := map[string]any{"job_id": 1, "equipment_id": 1, "quantity": 2, "days_used": 3, "notes": "Synthetic rental note"}
	_, p := prepare(map[string]any{"job_id": 1, "equipment_id": 1})
	required(p, "quantity")
	required(p, "days_used")
	_, p = prepare(map[string]any{"job_id": 1, "equipment_id": 3, "quantity": 1, "days_used": 1})
	required(p, "catalog_rental_price")
	_, p = prepare(map[string]any{"job_id": 1, "equipment_id": 5, "quantity": 2, "days_used": 2})
	required(p, "bounded_total_cost")
	for _, change := range []struct{ sql, guard string }{
		{`UPDATE rental_equipment SET rental_price=13.35 WHERE id=1`, "expected_context"},
		{`UPDATE rental_equipment SET is_active=false WHERE id=1`, "active_equipment"},
		{`UPDATE jobs SET description='Native header edit',updated_at=clock_timestamp() WHERE jobid=1`, "expected_job_updated_at"},
		{`UPDATE jobs SET deleted_at=clock_timestamp() WHERE jobid=1`, "restore_job_first"},
	} {
		stale, _ := prepare(base)
		exec(change.sql)
		status, out := invoke(stale, "rental-assignment-stale-"+change.guard)
		if status != 200 || count("job_rental_equipment") != 0 || count("rental_mcp_mutation_receipts") != 0 {
			t.Fatal("stale assignment was written", status, out)
		}
		required(out, change.guard)
		if change.guard == "active_equipment" {
			exec(`UPDATE rental_equipment SET is_active=true WHERE id=1`)
		}
		if change.guard == "restore_job_first" {
			exec(`UPDATE jobs SET deleted_at=NULL WHERE jobid=1`)
		}
	}
	exec(`UPDATE rental_equipment SET rental_price=12.35 WHERE id=1;INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	_, p = prepare(base)
	required(p, "active_editing_session")
	exec(`DELETE FROM job_edit_sessions`)
	in, p := prepare(base)
	if p["ready_to_execute"] != true || p["draft"].(map[string]any)["total_cost"] != float64(74.10) || count("job_rental_equipment") != 0 || count("audit_log") != 0 || count("job_history") != 0 {
		t.Fatal("preview pricing or side effect", p)
	}
	wrongPhrase := map[string]any{}
	for k, v := range in {
		wrongPhrase[k] = v
	}
	wrongPhrase["confirmation_text"] = "YES"
	if status, _ := invoke(wrongPhrase, "rental-assignment-bad-phrase"); status != 428 {
		t.Fatal("unbound confirmation", status)
	}
	if status, _ := invoke(in, ""); status != 428 {
		t.Fatal("missing idempotency key", status)
	}
	// A final audit failure must roll back the assignment, job version and receipt.
	exec(`CREATE FUNCTION fail_rental_assignment_audit() RETURNS TRIGGER LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'test audit failure';END;$$;CREATE TRIGGER fail_rental_assignment_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_rental_assignment_audit()`)
	if status, out := invoke(in, "rental-assignment-rollback"); status < 400 || count("job_rental_equipment") != 0 || count("job_history") != 0 || count("rental_mcp_mutation_receipts") != 0 {
		t.Fatal("partial rollback", status, out)
	}
	var version string
	if err := db.QueryRow(`SELECT to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM jobs WHERE jobid=1`).Scan(&version); err != nil || version != in["expected_job_updated_at"] {
		t.Fatal("job version changed on rollback", version, err)
	}
	exec(`DROP TRIGGER fail_rental_assignment_audit ON audit_log`)
	status, created := invoke(in, "rental-assignment-rollback")
	if status != 201 || count("job_rental_equipment") != 1 || count("job_history") != 1 || count("audit_log") != 1 || count("rental_mcp_mutation_receipts") != 1 {
		t.Fatal("atomic assignment", status, created)
	}
	status, replay := invoke(in, "rental-assignment-rollback")
	if status != 201 || !reflect.DeepEqual(created, replay) || count("job_rental_equipment") != 1 || count("audit_log") != 1 {
		t.Fatal("non-durable replay", status, replay)
	}
	wrongPhrase["confirmation_text"] = in["confirmation_text"]
	wrongPhrase["quantity"] = 3
	if status, _ := invoke(wrongPhrase, "rental-assignment-rollback"); status != 409 {
		t.Fatal("idempotency conflict", status)
	}
	financial = false
	if status, _ := invoke(in, "rental-assignment-rollback"); status != 403 {
		t.Fatal("financial replay bypass", status)
	}
	financial, action = true, "update"
	if status, _ := invoke(in, "rental-assignment-rollback"); status != 403 {
		t.Fatal("action replay bypass", status)
	}
	action, origin = "create", "other"
	if status, _ := invoke(in, "rental-assignment-rollback"); status != 403 {
		t.Fatal("origin bypass", status)
	}
	origin, actor = "MCP/AI", 2
	if status, _ := invoke(in, "rental-assignment-rollback"); status != 403 {
		t.Fatal("identity bypass", status)
	}
	actor = 1
	for _, revoked := range []string{"is_admin", "is_active"} {
		exec(`UPDATE users SET ` + revoked + `=false WHERE userid=1`)
		if status, _ := invoke(in, "rental-assignment-rollback"); status != 403 {
			t.Fatal("current rights replay bypass", revoked, status)
		}
		exec(`UPDATE users SET ` + revoked + `=true WHERE userid=1`)
	}
	duplicate, p := prepare(base)
	required(p, "equipment_already_assigned")
	if status, out := invoke(duplicate, "rental-assignment-duplicate"); status != 200 || out["operation_status"] != "needs_input" || count("job_rental_equipment") != 1 {
		t.Fatal("duplicate overwritten", status, out)
	}
	var quantity, days int
	var cost, revenue, final float64
	if err = db.QueryRow(`SELECT quantity,days_used,total_cost FROM job_rental_equipment WHERE job_id=1 AND equipment_id=1`).Scan(&quantity, &days, &cost); err != nil || quantity != 2 || days != 3 || cost != 74.10 {
		t.Fatal("wrong stored assignment", quantity, days, cost, err)
	}
	if err = db.QueryRow(`SELECT revenue,final_revenue FROM jobs WHERE jobid=1`).Scan(&revenue, &final); err != nil || revenue != 500 || final != 450 || count("job_positions") != 0 || count("job_product_requirements") != 0 {
		t.Fatal("assignment changed commercial positions/material", revenue, final, err)
	}
	exec(`UPDATE jobs SET multiply_by_days=false WHERE jobid=1`)
	single, p := prepare(map[string]any{"job_id": 1, "equipment_id": 2, "quantity": 2, "days_used": 10})
	if p["draft"].(map[string]any)["total_cost"] != float64(15) {
		t.Fatal("job day setting ignored", p)
	}
	if status, out := invoke(single, "rental-assignment-single-price"); status != 201 {
		t.Fatal(status, out)
	}
	free, p := prepare(map[string]any{"job_id": 1, "equipment_id": 4, "quantity": 1, "days_used": 1})
	if p["ready_to_execute"] != true || p["draft"].(map[string]any)["total_cost"] != float64(0) {
		t.Fatal("explicit catalog zero price rejected", p)
	}
	if status, out := invoke(free, "rental-assignment-zero-price"); status != 201 {
		t.Fatal(status, out)
	}
	for _, raw := range []string{
		`{"job_id":1,"equipment_id":1,"quantity":0,"days_used":1}`,
		`{"job_id":1,"equipment_id":1,"quantity":1.5,"days_used":1}`,
		`{"job_id":1,"equipment_id":1,"quantity":1001,"days_used":1}`,
		`{"job_id":1,"equipment_id":1,"quantity":1,"days_used":0}`,
		`{"job_id":1,"equipment_id":1,"quantity":1,"days_used":366}`,
		`{"job_id":-1,"equipment_id":1}`, `{"job_id":2147483648,"equipment_id":1}`,
		`{"job_id":1,"equipment_id":1,"rental_price":0}`, `{"job_id":1,"equipment_id":1,"total_cost":0}`,
		`{"job_id":1,"equipment_id":1,"product_id":1}`, `{} {}`,
	} {
		if status, _ := invokeRaw([]byte(raw), ""); status != 400 {
			t.Fatal("invalid or injected fields", status, raw)
		}
	}
	if status, _ := invoke(map[string]any{"notes": strings.Repeat("n", 501)}, ""); status != 400 {
		t.Fatal("overlong notes", status)
	}
}
