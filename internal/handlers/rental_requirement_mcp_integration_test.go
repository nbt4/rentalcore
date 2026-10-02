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

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/schema"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRentalRequirementOwnerAtomicSourcesAndLifecycle(t *testing.T) {
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
	fail := func(q string) {
		t.Helper()
		if _, err := db.Exec(q); err == nil {
			t.Fatal("unchecked native writer accepted", q)
		}
	}
	const ns = "rental_requirement_owner_test"
	exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE;CREATE SCHEMA ` + ns + `;SET search_path TO ` + ns)
	defer db.Exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE`)
	fixture, err := os.ReadFile("testdata/rental_job_mcp.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(fixture))
	exec(`CREATE TABLE products(productid SERIAL PRIMARY KEY,name TEXT,lifecycle_status TEXT DEFAULT 'active',updated_at TIMESTAMP DEFAULT NOW());INSERT INTO products(productid,name) VALUES(1,'Requirement test product'),(2,'Other requirement product');
ALTER TABLE devices ADD COLUMN lifecycle_status TEXT DEFAULT 'active';UPDATE devices SET productid=1;
ALTER TABLE job_positions ADD COLUMN product_id INT;ALTER TABLE job_positions ADD COLUMN position_type TEXT DEFAULT 'product';
ALTER TABLE job_product_requirements ADD COLUMN manual_quantity INT DEFAULT 0;ALTER TABLE job_product_requirements ADD COLUMN position_quantity INT DEFAULT 0;ALTER TABLE job_product_requirements ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE job_product_requirements ADD CONSTRAINT requirement_sources CHECK(quantity>0 AND manual_quantity>=0 AND position_quantity>=0 AND quantity=manual_quantity+position_quantity);
INSERT INTO jobs(jobid,job_code,customerid,statusid,description) VALUES(1,'JOB000001',1,1,'Requirement owner test');`)
	if err = schema.EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = schema.EnsureRentalRequirementLifecycle(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"../../migrations/049_rental_requirement_lifecycle.sql", "../../../migrations/postgresql/035_rental_requirement_lifecycle.sql"} {
		contents, err := os.ReadFile(path)
		if os.IsNotExist(err) && strings.HasPrefix(path, "../../../") {
			continue
		}
		if err != nil || strings.TrimSpace(string(contents)) != strings.TrimSpace(schema.RentalRequirementLifecycleSQL) {
			t.Fatal("migration mirror mismatch", path, err)
		}
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("requirement-test-", 4))
	gin.SetMode(gin.TestMode)
	h := &RentalRequirementMCP{db: db}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	router.POST("/requirements/:operation", h.Change)
	invoke := func(op string, in map[string]any, key string) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(in)
		req := httptest.NewRequest("POST", "/requirements/"+op, bytes.NewReader(raw))
		req.Header.Set("X-Cores-Origin", "MCP/AI")
		req.Header.Set("Idempotency-Key", key)
		action := op
		if op == "restore" {
			action = "archive"
		}
		tok, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": "cores:rental:" + action, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		if e != nil {
			t.Fatal(e)
		}
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: tok})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		out := map[string]any{}
		if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	prepare := func(op string, in map[string]any) (map[string]any, map[string]any) {
		t.Helper()
		copy := map[string]any{}
		for key, value := range in {
			copy[key] = value
		}
		copy["preview"] = true
		s, p := invoke(op, copy, "")
		if s != 200 {
			t.Fatalf("preview %d %#v", s, p)
		}
		delete(copy, "preview")
		copy["expected_updated_at"] = p["expected_updated_at"]
		copy["expected_job_updated_at"] = p["expected_job_updated_at"]
		copy["expected_context"] = p["expected_context"]
		copy["confirmation_text"] = p["required_confirmation_text"]
		copy["confirm_change"] = true
		return copy, p
	}
	execute := func(op string, in map[string]any, key string) map[string]any {
		t.Helper()
		s, out := invoke(op, in, key)
		if s < 200 || s > 299 || out["requirement"] == nil {
			t.Fatalf("execute %d %#v", s, out)
		}
		return out
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
		if e := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	exec(`INSERT INTO job_positions(job_id,product_id,position_type,quantity) VALUES(1,1,'product',2)`)
	in, p := prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 5, "manual_quantity": 3})
	if p["ready_to_execute"] != true || p["position_quantity"] != float64(2) {
		t.Fatal(p)
	}
	created := execute("create", in, "requirement-source-create")
	record := created["requirement"].(map[string]any)
	id := int(record["requirement_id"].(float64))
	if record["quantity"] != float64(5) || record["manual_quantity"] != float64(3) || record["position_quantity"] != float64(2) {
		t.Fatal(record)
	}
	s, replay := invoke("create", in, "requirement-source-create")
	if s != 201 || !reflect.DeepEqual(created, replay) || count("job_product_requirements") != 1 || count("audit_log") != 1 || count("job_history") != 1 {
		t.Fatal("non-atomic create/replay", s, replay)
	}
	exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ = invoke("create", in, "requirement-source-create"); s != 403 {
		t.Fatal("cached rights bypass", s)
	}
	exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	_, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 6})
	required(p, "requirement_already_exists")
	_, p = prepare("update", map[string]any{"requirement_id": id, "quantity": 1})
	required(p, "quantity_not_below_positions")
	_, p = prepare("update", map[string]any{"requirement_id": id, "quantity": 5, "manual_quantity": 4})
	required(p, "consistent_quantity_sources")
	in, p = prepare("update", map[string]any{"requirement_id": id, "manual_quantity": 0})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	updated := execute("update", in, "requirement-zero-manual")
	if updated["requirement"].(map[string]any)["quantity"] != float64(2) {
		t.Fatal(updated)
	}
	_, p = prepare("archive", map[string]any{"requirement_id": id})
	required(p, "commercial_position_contributions")
	exec(`DELETE FROM job_positions WHERE job_id=1`)
	in, p = prepare("update", map[string]any{"requirement_id": id, "manual_quantity": 3})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	execute("update", in, "requirement-manual-only")
	stale, _ := prepare("update", map[string]any{"requirement_id": id, "quantity": 4})
	exec(`UPDATE jobs SET description='Legacy job edit' WHERE jobid=1`)
	_, p = invoke("update", stale, "requirement-stale-parent")
	required(p, "expected_job_updated_at")
	required(p, "expected_context")
	stale, _ = prepare("update", map[string]any{"requirement_id": id, "quantity": 4})
	exec(`UPDATE job_product_requirements SET quantity=5,manual_quantity=5,updated_at='2000-01-01' WHERE id=$1`, id)
	_, p = invoke("update", stale, "requirement-stale-line")
	required(p, "expected_updated_at")
	required(p, "expected_context")
	stale, _ = prepare("update", map[string]any{"requirement_id": id, "quantity": 4})
	exec(`UPDATE products SET updated_at=clock_timestamp() WHERE productid=1`)
	_, p = invoke("update", stale, "requirement-stale-product")
	required(p, "expected_context")
	exec(`INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	_, p = prepare("update", map[string]any{"requirement_id": id, "quantity": 4})
	required(p, "active_editing_session")
	exec(`DELETE FROM job_edit_sessions`)
	exec(`INSERT INTO job_devices(jobid,deviceid) VALUES(1,'JOB-TEST-DEVICE')`)
	_, p = prepare("archive", map[string]any{"requirement_id": id})
	required(p, "assigned_equipment")
	exec(`UPDATE job_devices SET pack_status='returned'`)
	in, p = prepare("archive", map[string]any{"requirement_id": id})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	archived := execute("archive", in, "requirement-archive")
	if archived["requirement"].(map[string]any)["quantity"] != float64(5) {
		t.Fatal("archive lost quantities", archived)
	}
	_, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 5})
	required(p, "restoration_required")
	fail(`DELETE FROM job_product_requirements`)
	fail(`UPDATE job_product_requirements SET quantity=6,manual_quantity=6`)
	fail(`UPDATE job_product_requirements SET deleted_at=NULL,quantity=6,manual_quantity=6`)
	exec(`UPDATE products SET lifecycle_status='archived' WHERE productid=1`)
	_, p = prepare("restore", map[string]any{"requirement_id": id})
	required(p, "active_product")
	exec(`UPDATE products SET lifecycle_status='active' WHERE productid=1`)
	in, p = prepare("restore", map[string]any{"requirement_id": id})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	restored := execute("restore", in, "requirement-restore")
	if restored["requirement"].(map[string]any)["requirement_id"] != float64(id) || restored["requirement"].(map[string]any)["quantity"] != float64(5) {
		t.Fatal(restored)
	}
	in, p = prepare("update", map[string]any{"requirement_id": id, "quantity": 7})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	before := count("rental_mcp_mutation_receipts")
	history := count("job_history")
	exec(`CREATE FUNCTION fail_requirement_audit() RETURNS TRIGGER LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'test audit failure';END;$$;CREATE TRIGGER fail_requirement_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_requirement_audit()`)
	s, p = invoke("update", in, "requirement-rollback-retry")
	if s < 400 || count("rental_mcp_mutation_receipts") != before || count("job_history") != history {
		t.Fatal("partial audit commit", s, p)
	}
	exec(`DROP TRIGGER fail_requirement_audit ON audit_log`)
	done := execute("update", in, "requirement-rollback-retry")
	exec(`UPDATE jobs SET deleted_at=clock_timestamp(),statusid=6 WHERE jobid=1`)
	s, replay = invoke("update", in, "requirement-rollback-retry")
	if s != 200 || !reflect.DeepEqual(done, replay) {
		t.Fatal("durable replay after later parent changes", s, replay)
	}
	_, p = prepare("update", map[string]any{"requirement_id": id, "quantity": 8})
	required(p, "restore_job_first")
	for _, bad := range []map[string]any{{"requirement_id": id, "position_quantity": 7}, {"requirement_id": id, "product_id": 2, "quantity": 7}, {"requirement_id": id, "source_id": 3}, {"requirement_id": id, "quantity": -1}} {
		if s, _ := invoke("update", bad, ""); s != 400 {
			t.Fatal("injected/identity/source field", s, bad)
		}
	}
}
