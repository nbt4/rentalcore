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
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRentalPositionAtomicMigrationPricingAndArchive(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	u, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("dedicated _test DB required")
	}
	db, e := sql.Open("pgx", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := db.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`DROP SCHEMA IF EXISTS rental_position_owner_test CASCADE;CREATE SCHEMA rental_position_owner_test;SET search_path TO rental_position_owner_test`)
	defer db.Exec(`DROP SCHEMA IF EXISTS rental_position_owner_test CASCADE`)
	fixture, e := os.ReadFile("testdata/rental_job_mcp.sql")
	if e != nil {
		t.Fatal(e)
	}
	exec(string(fixture))
	exec(`CREATE TABLE products(productid SERIAL PRIMARY KEY,name TEXT,lifecycle_status TEXT DEFAULT 'active',updated_at TIMESTAMP DEFAULT NOW());INSERT INTO products VALUES(1,'Test product','active',NOW()),(2,'Other product','active',NOW());
 ALTER TABLE devices ADD COLUMN lifecycle_status TEXT DEFAULT 'active';UPDATE devices SET productid=1;
 ALTER TABLE job_positions ADD COLUMN product_id INT;ALTER TABLE job_positions ADD COLUMN position_type TEXT DEFAULT 'product';
 ALTER TABLE job_positions ADD COLUMN service_item_id BIGINT;ALTER TABLE job_positions ADD COLUMN rental_equipment_id BIGINT;ALTER TABLE job_positions ADD COLUMN pdf_extraction_item_id BIGINT;
 ALTER TABLE job_positions ADD COLUMN description TEXT DEFAULT '';ALTER TABLE job_positions ADD COLUMN unit TEXT DEFAULT 'Stück';ALTER TABLE job_positions ADD COLUMN sort_order INT DEFAULT 0;ALTER TABLE job_positions ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
 ALTER TABLE job_product_requirements ADD COLUMN manual_quantity INT DEFAULT 0;ALTER TABLE job_product_requirements ADD COLUMN position_quantity INT DEFAULT 0;ALTER TABLE job_product_requirements ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();
 ALTER TABLE job_product_requirements ADD CONSTRAINT requirement_identity UNIQUE(job_id,product_id);
 ALTER TABLE job_product_requirements ADD CONSTRAINT requirement_sources CHECK(quantity>0 AND manual_quantity>=0 AND position_quantity>=0 AND quantity=manual_quantity+position_quantity);
 INSERT INTO jobs(jobid,job_code,customerid,statusid,description,startdate,enddate,discount,discount_type) VALUES(1,'JOB001',1,1,'Position owner test','2026-10-01','2026-10-04',10,'percent');`)
	for _, ensure := range []func(*sql.DB) error{schema.EnsureRentalJobLifecycle, schema.EnsureRentalPositionLifecycle, schema.EnsureRentalRequirementLifecycle, schema.EnsureRentalPositionLifecycle} {
		if e := ensure(db); e != nil {
			t.Fatal(e)
		}
	}
	for _, path := range []string{"../../migrations/050_rental_position_lifecycle.sql", "../../../migrations/postgresql/047_rental_position_lifecycle.sql"} {
		raw, e := os.ReadFile(path)
		if os.IsNotExist(e) && strings.HasPrefix(path, "../../../") {
			continue
		}
		if e != nil || strings.TrimSpace(string(raw)) != strings.TrimSpace(schema.RentalPositionLifecycleSQL) {
			t.Fatal("migration mirror mismatch", path, e)
		}
	}
	orm, e := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	h := NewRentalPositionMCP(&repository.Database{DB: orm})
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("position-test-", 4))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	router.POST("/positions/:operation", h.Change)
	financial := true
	delegatedAction := ""
	invoke := func(op string, in map[string]any, key string) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(in)
		req := httptest.NewRequest("POST", "/positions/"+op, bytes.NewReader(raw))
		req.Header.Set("X-Cores-Origin", "MCP/AI")
		req.Header.Set("Idempotency-Key", key)
		action := op
		if delegatedAction != "" {
			action = delegatedAction
		}
		tok, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": "cores:rental:" + action, "mcp_financial": financial, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
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
		for k, v := range in {
			copy[k] = v
		}
		copy["preview"] = true
		status, p := invoke(op, copy, "")
		if status != 200 {
			t.Fatal(status, p)
		}
		delete(copy, "preview")
		if p["expected_updated_at"] != nil {
			copy["expected_updated_at"] = p["expected_updated_at"]
		}
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
	execute := func(op string, in map[string]any, key string) map[string]any {
		t.Helper()
		s, p := invoke(op, in, key)
		if s < 200 || s > 299 || p["position"] == nil {
			t.Fatal("execute", s, p)
		}
		return p
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if e := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	assertState := func(total, manual, source int, revenue, final float64) {
		t.Helper()
		var q, m, p int
		var r, f float64
		if e := db.QueryRow(`SELECT quantity,manual_quantity,position_quantity FROM job_product_requirements WHERE job_id=1 AND product_id=1 AND deleted_at IS NULL`).Scan(&q, &m, &p); e != nil {
			t.Fatal(e)
		}
		if e := db.QueryRow(`SELECT revenue,final_revenue FROM jobs WHERE jobid=1`).Scan(&r, &f); e != nil {
			t.Fatal(e)
		}
		if q != total || m != manual || p != source || r != revenue || f != final {
			t.Fatal("wrong material/job value", q, m, p, r, f)
		}
	}
	exec(`INSERT INTO job_product_requirements(job_id,product_id,quantity,manual_quantity) VALUES(1,1,9,9)`)
	var requirementID int
	db.QueryRow(`SELECT id FROM job_product_requirements WHERE job_id=1 AND product_id=1`).Scan(&requirementID)
	_, p := prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 9})
	required(p, "unit_price")
	required(p, "manual_quantity_or_preserve_manual_quantity")
	in, p := prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 9, "unit_price": 10, "manual_quantity": 0})
	if p["ready_to_execute"] != true || p["effects"].(map[string]any)["revenue_after"] != float64(214.2) || count("job_positions") != 0 {
		t.Fatal("preview effects/writes", p)
	}
	created := execute("create", in, "position-migrate-manual")
	record := created["position"].(map[string]any)
	id := int(record["position_id"].(float64))
	assertState(9, 0, 9, 214.2, 192.78)
	var retained int
	db.QueryRow(`SELECT id FROM job_product_requirements WHERE job_id=1 AND product_id=1`).Scan(&retained)
	if retained != requirementID {
		t.Fatal("lost requirement identity")
	}
	s, replay := invoke("create", in, "position-migrate-manual")
	if s != 201 || !reflect.DeepEqual(created, replay) || count("job_positions") != 1 || count("audit_log") != 1 || count("job_history") != 1 {
		t.Fatal("non-atomic replay", s, replay)
	}
	conflicting := map[string]any{}
	for k, v := range in {
		conflicting[k] = v
	}
	conflicting["quantity"] = 10
	if s, _ := invoke("create", conflicting, "position-migrate-manual"); s != 409 {
		t.Fatal("key conflict", s)
	}
	financial = false
	if s, _ := invoke("create", in, "position-migrate-manual"); s != 403 {
		t.Fatal("financial replay bypass", s)
	}
	financial = true
	exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ := invoke("create", in, "position-migrate-manual"); s != 403 {
		t.Fatal("rights replay bypass", s)
	}
	exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	delegatedAction = "create"
	if s, _ := invoke("archive", map[string]any{"position_id": id}, ""); s != 403 {
		t.Fatal("action scope bypass")
	}
	delegatedAction = ""
	_, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": 0})
	required(p, "review_duplicate_positions")
	_, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": 0, "allow_duplicate": true})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	// Changed product, parent, native line, assignments, and editors invalidate preparation.
	for _, change := range []struct{ sql, guard string }{{`UPDATE products SET updated_at=clock_timestamp() WHERE productid=1`, "expected_context"}, {`UPDATE jobs SET description='Native header edit' WHERE jobid=1`, "expected_job_updated_at"}, {`UPDATE job_positions SET unit_price=11 WHERE position_id=1`, "expected_updated_at"}} {
		stale, _ := prepare("update", map[string]any{"position_id": id, "quantity": 8})
		exec(change.sql)
		_, p = invoke("update", stale, "position-stale-"+change.guard)
		required(p, change.guard)
	}
	exec(`INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	_, p = prepare("update", map[string]any{"position_id": id, "quantity": 8})
	required(p, "active_editing_session")
	exec(`DELETE FROM job_edit_sessions`)
	in, p = prepare("update", map[string]any{"position_id": id, "quantity": 8, "unit_price": 10, "tax_rate": 0, "discount_percent": 10, "discount_amount": 4})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	execute("update", in, "position-discount-update")
	assertState(8, 0, 8, 140, 126)
	exec(`INSERT INTO job_devices(jobid,deviceid) VALUES(1,'JOB-TEST-DEVICE')`)
	_, p = prepare("archive", map[string]any{"position_id": id})
	required(p, "quantity_not_below_assigned_devices")
	exec(`INSERT INTO job_position_devices(position_id,device_id) VALUES($1,'JOB-TEST-DEVICE')`, id)
	_, p = prepare("archive", map[string]any{"position_id": id})
	required(p, "assigned_position_devices")
	exec(`DELETE FROM job_position_devices;UPDATE job_devices SET pack_status='returned'`)
	// Failure in the final audit rolls back position, sources, revenue, history and receipt.
	in, p = prepare("archive", map[string]any{"position_id": id})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	receipts, history := count("rental_mcp_mutation_receipts"), count("job_history")
	exec(`CREATE FUNCTION fail_position_audit() RETURNS TRIGGER LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'test audit failure';END;$$;CREATE TRIGGER fail_position_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_position_audit()`)
	s, p = invoke("archive", in, "position-archive-rollback")
	if s < 400 || count("rental_mcp_mutation_receipts") != receipts || count("job_history") != history {
		t.Fatal("partial rollback", s, p)
	}
	assertState(8, 0, 8, 140, 126)
	exec(`DROP TRIGGER fail_position_audit ON audit_log`)
	archived := execute("archive", in, "position-archive-rollback")
	if archived["position"].(map[string]any)["quantity"] != float64(8) {
		t.Fatal("archive lost fields", archived)
	}
	var live, archivedReq int
	var revenue float64
	db.QueryRow(`SELECT count(*) FROM job_positions WHERE deleted_at IS NULL`).Scan(&live)
	db.QueryRow(`SELECT count(*) FROM job_product_requirements WHERE deleted_at IS NOT NULL`).Scan(&archivedReq)
	db.QueryRow(`SELECT revenue FROM jobs WHERE jobid=1`).Scan(&revenue)
	if live != 0 || archivedReq != 1 || revenue != 0 {
		t.Fatal("archive left demand/revenue", live, archivedReq, revenue)
	}
	if _, e = db.Exec(`INSERT INTO job_position_devices(position_id,device_id) VALUES($1,'JOB-TEST-DEVICE')`, id); e == nil {
		t.Fatal("assignment to archived position accepted")
	}
	if _, e = db.Exec(`DELETE FROM job_positions`); e == nil {
		t.Fatal("hard-delete accepted")
	}
	if _, e = db.Exec(`UPDATE job_positions SET description='edit archive'`); e == nil {
		t.Fatal("archived edit accepted")
	}
	_, p = prepare("update", map[string]any{"position_id": id, "quantity": 2})
	required(p, "active_position")
	// Native reconciliation and read paths exclude the retained archive.
	repo := repository.NewPositionRepository(&repository.Database{DB: orm})
	positions, e := repo.GetByJobID(1)
	if e != nil || len(positions) != 0 {
		t.Fatal("archive in native UI", positions, e)
	}
	in, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 2, "unit_price": 0})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	execute("create", in, "position-reuse-requirement")
	assertState(2, 0, 2, 0, 0)
	exec(`UPDATE job_product_requirements SET quantity=5,manual_quantity=3 WHERE job_id=1 AND product_id=1`)
	_, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": 0, "allow_duplicate": true})
	required(p, "manual_quantity_or_preserve_manual_quantity")
	in, p = prepare("create", map[string]any{"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": 0, "allow_duplicate": true, "preserve_manual_quantity": true})
	extra := execute("create", in, "position-preserve-manual")
	assertState(6, 3, 3, 0, 0)
	extraID := extra["position"].(map[string]any)["position_id"]
	in, p = prepare("archive", map[string]any{"position_id": extraID})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	execute("archive", in, "position-preserve-on-archive")
	assertState(5, 3, 2, 0, 0)

	for _, bad := range []map[string]any{{"job_id": 1, "product_id": 1, "quantity": 1.5, "unit_price": 0}, {"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": 0.001}, {"job_id": 1, "product_id": 1, "quantity": 1, "unit_price": -1}, {"job_id": 1, "product_id": 1, "position_type": "service"}, {"position_id": id, "product_id": 2, "quantity": 1}, {"position_id": id, "quantity": 1, "manual_quantity": -1}} {
		op := "create"
		if bad["position_id"] != nil {
			op = "update"
		}
		if s, _ := invoke(op, bad, ""); s != 400 {
			t.Fatal("invalid/injected fields", s, bad)
		}
	}
	// Re-finalizing the same PDF retains archived provenance without colliding
	// with its active unique identity or duplicating live demand/revenue.
	exec(`INSERT INTO jobs(jobid,job_code,customerid,statusid,description) VALUES(2,'JOB002',1,1,'PDF refresh fixture');
 CREATE TABLE pdf_extraction_items(item_id BIGINT PRIMARY KEY,extraction_id BIGINT,raw_product_text TEXT,quantity INT,unit_price NUMERIC,mapped_product_id INT,mapping_status TEXT);
 INSERT INTO pdf_extraction_items VALUES(101,100,'PDF product',3,10,2,'user_confirmed')`)
	pdf := &PDFHandler{DB: orm, JobHandler: &JobHandler{requirementRepo: repository.NewRequirementRepository(&repository.Database{DB: orm})}}
	if e = pdf.createPositionsFromExtraction(&models.Job{JobID: 2}, 100); e != nil {
		t.Fatal("first PDF finalize", e)
	}
	exec(`UPDATE pdf_extraction_items SET quantity=5 WHERE item_id=101`)
	if e = pdf.createPositionsFromExtraction(&models.Job{JobID: 2}, 100); e != nil {
		t.Fatal("repeat PDF finalize", e)
	}
	var pdfActive, pdfArchived, pdfDemand int
	var pdfRevenue float64
	db.QueryRow(`SELECT count(*) FILTER(WHERE deleted_at IS NULL),count(*) FILTER(WHERE deleted_at IS NOT NULL) FROM job_positions WHERE job_id=2 AND pdf_extraction_item_id=101`).Scan(&pdfActive, &pdfArchived)
	db.QueryRow(`SELECT quantity FROM job_product_requirements WHERE job_id=2 AND product_id=2 AND deleted_at IS NULL`).Scan(&pdfDemand)
	db.QueryRow(`SELECT revenue FROM jobs WHERE jobid=2`).Scan(&pdfRevenue)
	if pdfActive != 1 || pdfArchived != 1 || pdfDemand != 5 || pdfRevenue != 59.5 {
		t.Fatal("PDF refresh duplicated demand/revenue", pdfActive, pdfArchived, pdfDemand, pdfRevenue)
	}

}
