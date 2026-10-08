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
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
 INSERT INTO rental_equipment(id,name,supplier,rental_price,customer_price) VALUES(1,'Test rental one','Test supplier',12.35,20),(2,'Test rental two','Test supplier',7.50,15),(3,'Missing catalog price','Test supplier',NULL,NULL),(4,'Test free rental','Test supplier',0,0),(5,'Test costly rental','Test supplier',9999999999.99,0),(6,'Missing customer price','Test supplier',37.50,NULL);
 INSERT INTO jobs(jobid,job_code,customerid,statusid,description,startdate,enddate,revenue,final_revenue) VALUES(1,'JOB_TEST_RENTAL',1,1,'Rental assignment fixture','2026-10-01','2026-10-04',0,0)`)
	assignmentMigration, err := os.ReadFile("../../migrations/039_job_rental_equipment.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(assignmentMigration))

	exec(`DROP TABLE job_positions;CREATE TABLE products(productid SERIAL PRIMARY KEY,name TEXT);CREATE TABLE service_items(id BIGSERIAL PRIMARY KEY,name TEXT)`)
	for _, path := range []string{"../../migrations/038_job_positions.up.sql", "../../migrations/040_job_positions_tax_rental.up.sql"} {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		exec(string(raw))
	}
	exec(`ALTER TABLE job_positions ADD COLUMN pdf_extraction_item_id BIGINT;ALTER TABLE job_positions ADD COLUMN deleted_at TIMESTAMPTZ;ALTER TABLE job_product_requirements ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();ALTER TABLE devices ADD COLUMN serialnumber TEXT`)
	if err = schema.EnsureRentalPositionLifecycle(db); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/051_rental_position_cost_link.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(migration))
	exec(string(migration))
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = schema.EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("external-rental-test-", 3))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	actor := uint(1)
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: actor}) })
	h := &RentalJobMCP{db: db, orm: orm}
	database := &repository.Database{DB: orm}
	positionRepo := repository.NewPositionRepository(database)
	ui := NewPositionHandler(positionRepo, repository.NewJobRepository(database), repository.NewRequirementRepository(database), orm)
	router.GET("/rental-catalog", ui.GetRentalCatalog)
	router.GET("/jobs/:id/positions", ui.GetPositions)
	router.POST("/jobs/:id/positions", ui.CreatePosition)
	router.PUT("/jobs/:id/positions/:posId", ui.UpdatePosition)
	router.DELETE("/jobs/:id/positions/:posId", ui.DeletePosition)
	router.PATCH("/jobs/:id/settings", ui.UpdatePriceSettings)
	router.GET("/analytics", NewAnalyticsHandler(orm).GetRevenueDrilldown)
	uiCall := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		out := map[string]any{}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	status, catalog := uiCall("GET", "/rental-catalog", "")
	if status != 200 {
		t.Fatal(status, catalog)
	}
	for _, value := range catalog["items"].([]any) {
		row := value.(map[string]any)
		if row["equipmentID"] == float64(1) && (row["rentalPrice"] != 12.35 || row["customerPrice"] != float64(20)) {
			t.Fatal("catalog prices conflated", row)
		}
	}

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
	required(p, "catalog_customer_price")
	_, p = prepare(map[string]any{"job_id": 1, "equipment_id": 6, "quantity": 1, "days_used": 1})
	required(p, "catalog_customer_price")
	_, p = prepare(map[string]any{"job_id": 1, "equipment_id": 5, "quantity": 2, "days_used": 2})
	required(p, "bounded_total_cost")
	for _, change := range []struct{ sql, guard string }{
		{`UPDATE rental_equipment SET rental_price=13.35 WHERE id=1`, "expected_context"},
		{`UPDATE rental_equipment SET customer_price=21 WHERE id=1`, "expected_context"},
		{`INSERT INTO job_positions(job_id,position_type,description,quantity,unit_price) VALUES(1,'service','Concurrent UI edit',1,10)`, "expected_context"},
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
	exec(`UPDATE rental_equipment SET rental_price=12.35,customer_price=20 WHERE id=1;UPDATE job_positions SET deleted_at=NOW();INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	_, p = prepare(base)
	required(p, "active_editing_session")
	exec(`DELETE FROM job_edit_sessions`)
	in, p := prepare(base)
	if p["ready_to_execute"] != true || p["draft"].(map[string]any)["total_cost"] != float64(74.10) || count("job_rental_equipment") != 0 || count("audit_log") != 0 || count("job_history") != 0 {
		t.Fatal("preview pricing or side effect", p)
	}
	effects := p["effects"].(map[string]any)
	if effects["customer_price"] != float64(20) || effects["rental_price"] != 12.35 || effects["customer_sales_net"] != float64(40) || effects["expected_margin"] != -34.10 || effects["revenue_after"] != 47.60 {
		t.Fatal("incomplete or wrong commercial preview", effects)
	}
	previewPosition := p["draft"].(map[string]any)["position"].(map[string]any)
	if previewPosition["line_net"] != float64(40) || previewPosition["line_gross"] != 47.60 {
		t.Fatal("preview line amounts absent", previewPosition)
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
	if status, out := invoke(in, "rental-assignment-rollback"); status < 400 || count("job_positions") != 1 || count("job_rental_equipment") != 0 || count("job_history") != 0 || count("rental_mcp_mutation_receipts") != 0 {
		t.Fatal("partial rollback", status, out)
	}
	var version string
	if err := db.QueryRow(`SELECT to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM jobs WHERE jobid=1`).Scan(&version); err != nil || version != in["expected_job_updated_at"] {
		t.Fatal("job version changed on rollback", version, err)
	}
	exec(`DROP TRIGGER fail_rental_assignment_audit ON audit_log`)
	status, created := invoke(in, "rental-assignment-rollback")
	if status != 201 || count("job_positions") != 2 || count("job_rental_equipment") != 1 || count("job_history") != 1 || count("audit_log") != 1 || count("rental_mcp_mutation_receipts") != 1 {
		t.Fatal("atomic assignment", status, created)
	}
	analyticsStatus, analytics := uiCall("GET", "/analytics?scope=pipeline", "")
	if analyticsStatus != 200 || analytics["rental_cost"] != 74.10 || analytics["rental_net_revenue"] != float64(40) || analytics["rental_margin"] != -34.10 {
		t.Fatal("actual analytics doubled costs or included tax in margin", analyticsStatus, analytics)
	}
	status, replay := invoke(in, "rental-assignment-rollback")
	if status != 201 || count("job_positions") != 2 || !reflect.DeepEqual(created, replay) || count("job_rental_equipment") != 1 || count("audit_log") != 1 {
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
	if err = db.QueryRow(`SELECT revenue,final_revenue FROM jobs WHERE jobid=1`).Scan(&revenue, &final); err != nil || revenue != 47.60 || final != 47.60 || count("job_positions") != 2 || count("job_product_requirements") != 0 {
		t.Fatal("assignment did not create the canonical commercial position", revenue, final, err)
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

	positions, e := positionRepo.GetByJobID(1)
	if e != nil || len(positions) != 3 {
		t.Fatal("MCP positions missing from native read", positions, e)
	}
	first := created["position"].(map[string]any)
	if first["position_type"] != "rental" || first["rental_equipment_id"] != float64(1) || first["quantity"] != float64(2) || first["unit_price"] != float64(20) || first["follow_day_factor"] != float64(0) {
		t.Fatal("wrong canonical position", first)
	}
	readStatus, read := uiCall("GET", "/jobs/1/positions", "")
	if readStatus != 200 || len(read["positions"].([]any)) != 3 {
		t.Fatal("normal job UI read", readStatus, read)
	}
	// A separate synthetic legacy job stands in for an orphan such as JOB001165.
	exec(`INSERT INTO jobs(jobid,job_code,customerid,statusid,description,startdate,enddate,multiply_by_days) VALUES(2,'REPAIR_TEST',1,1,'Synthetic repair','2026-10-01','2026-10-04',true),(3,'UI_TEST',1,1,'Synthetic UI','2026-10-01','2026-10-04',true);
      INSERT INTO job_rental_equipment(job_id,equipment_id,quantity,days_used,total_cost,notes) VALUES(2,1,2,3,60,'Historic note');
      INSERT INTO job_history(job_id,description) VALUES(2,'Historic MCP assignment')`)
	// The migration must be compatible with the old UI handler during rollout/rollback.
	exec(`INSERT INTO jobs(jobid,job_code,customerid,statusid,description,multiply_by_days) VALUES(5,'OLD_HANDLER_TEST',1,1,'Synthetic old handler',true);
        INSERT INTO job_rental_equipment(job_id,equipment_id,quantity,days_used,total_cost) VALUES(5,1,2,3,225),(5,4,1,3,0)`)
	exec(`UPDATE jobs SET multiply_by_days=false WHERE jobid=5;
        UPDATE job_rental_equipment SET total_cost=round(total_cost/GREATEST(days_used,1)::numeric,2) WHERE job_id=5`)
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=5 AND equipment_id=1`).Scan(&cost); err != nil || cost != 75 {
		t.Fatal("old handler applied day divisor twice", cost, err)
	}
	exec(`UPDATE jobs SET multiply_by_days=true WHERE jobid=5;
        UPDATE job_rental_equipment SET total_cost=round(total_cost*GREATEST(days_used,1)::numeric,2) WHERE job_id=5`)
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=5 AND equipment_id=1`).Scan(&cost); err != nil || cost != 225 {
		t.Fatal("old handler applied day multiplier twice", cost, err)
	}
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=5 AND equipment_id=4`).Scan(&cost); err != nil || cost != 0 {
		t.Fatal("old handler invented zero-price costs", cost, err)
	}
	exec(`UPDATE jobs SET statusid=6 WHERE jobid=5`)
	beforeHistory := count("job_history")
	repairBase := map[string]any{"job_id": 2, "equipment_id": 1, "quantity": 2, "days_used": 3, "repair_existing": true}
	var ledgerBefore, ledgerAfter string
	if err := db.QueryRow(`SELECT jsonb_agg(to_jsonb(c) ORDER BY job_id,equipment_id)::text FROM job_rental_equipment c`).Scan(&ledgerBefore); err != nil {
		t.Fatal(err)
	}
	exec(string(migration))
	if err := db.QueryRow(`SELECT jsonb_agg(to_jsonb(c) ORDER BY job_id,equipment_id)::text FROM job_rental_equipment c`).Scan(&ledgerAfter); err != nil || ledgerBefore != ledgerAfter {
		t.Fatal("migration changed legacy costs", err)
	}
	// A pre-existing inconsistent snapshot must block repair without changing stored cost.
	exec(`ALTER TABLE job_rental_equipment DISABLE TRIGGER normalize_job_rental_captured_cost;
        UPDATE job_rental_equipment SET rental_unit_price=99 WHERE job_id=2;
        ALTER TABLE job_rental_equipment ENABLE TRIGGER normalize_job_rental_captured_cost`)
	_, inconsistent := prepare(map[string]any{"job_id": 2, "equipment_id": 1, "quantity": 2, "days_used": 3, "repair_existing": true})
	required(inconsistent, "consistent_existing_supplier_cost")
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=2`).Scan(&cost); err != nil || cost != 60 {
		t.Fatal("inconsistent repair changed old cost", cost, err)
	}
	exec(`UPDATE job_rental_equipment SET rental_unit_price=NULL WHERE job_id=2`)
	_, wrongRepair := prepare(map[string]any{"job_id": 2, "equipment_id": 1, "quantity": 3, "days_used": 3, "repair_existing": true})
	required(wrongRepair, "exact_existing_quantity_and_days_required")
	_, wrongNotes := prepare(map[string]any{"job_id": 2, "equipment_id": 1, "quantity": 2, "days_used": 3, "repair_existing": true, "notes": "replace historic note"})
	required(wrongNotes, "preserve_existing_assignment_notes")
	repair, p := prepare(repairBase)
	if p["ready_to_execute"] != true || p["effects"].(map[string]any)["rental_cost_after"] != float64(60) {
		t.Fatal("legacy repair preview", p)
	}
	// Changing a captured cost without touching the job timestamp invalidates repair.
	exec(`UPDATE job_rental_equipment SET total_cost=61 WHERE job_id=2`)
	if code, out := invoke(repair, "repair-stale-context"); code != 200 || out["ready_to_execute"] != false {
		t.Fatal("stale repair wrote", code, out)
	}
	exec(`UPDATE job_rental_equipment SET total_cost=60 WHERE job_id=2`)
	repair, p = prepare(repairBase)
	rowsBefore := count("job_rental_equipment")
	posBefore := count("job_positions")
	code, repaired := invoke(repair, "repair-legacy-position")
	if code != 201 || count("job_rental_equipment") != rowsBefore || count("job_positions") != posBefore+1 || count("job_history") != beforeHistory+1 {
		t.Fatal("non-atomic or duplicate repair", code, repaired)
	}
	code, replayed := invoke(repair, "repair-legacy-position")
	if code != 201 || !reflect.DeepEqual(repaired, replayed) || count("job_positions") != posBefore+1 {
		t.Fatal("repair replay duplicated position", code, replayed)
	}
	_, p = prepare(repairBase)
	required(p, "assignment_already_linked")
	required(p, "rental_position_already_exists")
	var preservedNotes string
	if err := db.QueryRow(`SELECT total_cost,notes FROM job_rental_equipment WHERE job_id=2`).Scan(&cost, &preservedNotes); err != nil || cost != 60 || preservedNotes != "Historic note" {
		t.Fatal("repair destroyed historic costs", cost, preservedNotes, err)
	}
	exec(`INSERT INTO jobs(jobid,job_code,customerid,statusid,description) VALUES(4,'CORRECT_EXISTING_TEST',1,1,'Synthetic existing position');
     INSERT INTO job_positions(job_id,position_type,rental_equipment_id,description,quantity,unit_price) VALUES(4,'rental',2,'Preserve agreed customer price',1,99);
     INSERT INTO job_rental_equipment(job_id,equipment_id,quantity,days_used,total_cost) VALUES(4,2,1,1,7.50)`)
	_, p = prepare(map[string]any{"job_id": 4, "equipment_id": 2, "quantity": 1, "days_used": 1, "repair_existing": true})
	required(p, "rental_position_already_exists")
	var agreedPrice float64
	if err := db.QueryRow(`SELECT unit_price FROM job_positions WHERE job_id=4`).Scan(&agreedPrice); err != nil || agreedPrice != 99 {
		t.Fatal("correct position overwritten", agreedPrice, err)
	}
	if _, err := db.Exec(`UPDATE job_rental_equipment SET position_id=(SELECT position_id FROM job_positions WHERE job_id=1 AND rental_equipment_id=4) WHERE job_id=1 AND equipment_id=1`); err == nil {
		t.Fatal("mismatched supplier cost link accepted")
	}
	// The API rejects missing prices even if the caller injects a unit price.
	code, out := uiCall("POST", "/jobs/3/positions", `{"position_type":"rental","rental_equipment_id":6,"quantity":1,"unit_price":37.5}`)
	if code != 400 {
		t.Fatal("missing customer price invented", code, out)
	}
	code, out = uiCall("POST", "/jobs/3/positions", `{"position_type":"rental","rental_equipment_id":1,"quantity":2,"unit_price":12.35}`)
	if code != 201 {
		t.Fatal("native UI creation", code, out)
	}
	nativePosition := out["position"].(map[string]any)
	if nativePosition["unit_price"] != float64(20) || nativePosition["follow_day_factor"] != float64(0) {
		t.Fatal("UI used purchase price or GORM default factor", nativePosition)
	}
	id := int(nativePosition["position_id"].(float64))
	var nativeDays int
	if err := db.QueryRow(`SELECT days_used,total_cost FROM job_rental_equipment WHERE job_id=3`).Scan(&nativeDays, &cost); err != nil || nativeDays != 3 || cost != 74.10 {
		t.Fatal("native supplier costs", nativeDays, cost, err)
	}
	code, out = uiCall("POST", "/jobs/3/positions", `{"position_type":"rental","rental_equipment_id":1,"quantity":2}`)
	if code != 409 {
		t.Fatal("native duplicate rental", code, out)
	}
	// Supplier catalog changes never reprice captured costs on subsequent edits.
	exec(`UPDATE rental_equipment SET rental_price=100 WHERE id=1`)
	code, out = uiCall("PUT", fmt.Sprintf("/jobs/3/positions/%d", id), `{"quantity":3}`)
	if code != 200 {
		t.Fatal(code, out)
	}
	if err := db.QueryRow(`SELECT quantity,total_cost FROM job_rental_equipment WHERE job_id=3`).Scan(&quantity, &cost); err != nil || quantity != 3 || cost != 111.15 {
		t.Fatal("quantity cost snapshot", quantity, cost, err)
	}
	code, out = uiCall("PATCH", "/jobs/3/settings", `{"multiply_by_days":false}`)
	if code != 200 {
		t.Fatal(code, out)
	}
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=3`).Scan(&cost); err != nil || cost != 37.05 {
		t.Fatal("flat costs", cost, err)
	}
	code, out = uiCall("PATCH", "/jobs/3/settings", `{"multiply_by_days":true}`)
	if code != 200 {
		t.Fatal(code, out)
	}
	if err := db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=3`).Scan(&cost); err != nil || cost != 111.15 {
		t.Fatal("day toggle drift", cost, err)
	}
	analyticsStatus, analyticsBefore := uiCall("GET", "/analytics?scope=pipeline", "")
	if analyticsStatus != 200 {
		t.Fatal(analyticsStatus, analyticsBefore)
	}
	// Costs/IDs remain retained after native archive, but leave current analytics.
	code, out = uiCall("DELETE", fmt.Sprintf("/jobs/3/positions/%d", id), "")
	if code != 200 {
		t.Fatal(code, out)
	}
	analyticsStatus, analyticsAfter := uiCall("GET", "/analytics?scope=pipeline", "")
	if analyticsStatus != 200 {
		t.Fatal(analyticsStatus, analyticsAfter)
	}
	assertClose(t, analyticsBefore["rental_cost"].(float64)-analyticsAfter["rental_cost"].(float64), 111.15)
	activeCosts := 0
	if err := db.QueryRow(`SELECT count(*) FROM job_rental_equipment c LEFT JOIN job_positions p ON p.position_id=c.position_id WHERE c.job_id=3 AND (c.position_id IS NULL OR p.deleted_at IS NULL)`).Scan(&activeCosts); err != nil || activeCosts != 0 {
		t.Fatal("archived costs still active", activeCosts, err)
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
