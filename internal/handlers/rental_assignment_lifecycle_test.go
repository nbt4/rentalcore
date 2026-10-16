package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAssignmentFixture(t *testing.T) (*externalCatalogFixture, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("isolated _test PostgreSQL required")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("dedicated _test database required")
	}
	sum := sha256.Sum256([]byte(t.Name()))
	ns := "rental_assignment_" + hex.EncodeToString(sum[:6])
	q := u.Query()
	q.Set("search_path", ns)
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { db.Exec("DROP SCHEMA " + ns + " CASCADE"); db.Close() })
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE SCHEMA " + ns)
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

	exec(`ALTER TABLE rental_equipment ADD COLUMN created_at TIMESTAMP DEFAULT NOW();ALTER TABLE rental_equipment ADD COLUMN supplier_id INTEGER;ALTER TABLE rental_equipment ADD COLUMN description TEXT;ALTER TABLE rental_equipment ADD COLUMN notes TEXT;ALTER TABLE customers ADD COLUMN is_supplier BOOLEAN DEFAULT true`)
	helper, err := os.ReadFile("testdata/external_catalog_lifecycle.sql")
	if err != nil {
		t.Fatal(err)
	}
	helperSQL := string(helper)
	i := strings.Index(helperSQL, "CREATE FUNCTION rental_master_job_is_closed")
	exec(helperSQL[i:])
	if err = schema.EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	if err = schema.EnsureRentalExternalCatalogLifecycle(db); err != nil {
		t.Fatal(err)
	}
	if err = schema.EnsureRentalAssignmentLifecycle(db); err != nil {
		t.Fatal(err)
	}
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("assignment-lifecycle-", 3))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	h := &RentalJobMCP{db: db, orm: orm}
	router.POST("/create", h.createExternalEquipment)
	router.POST("/:operation", func(c *gin.Context) { h.changeExternalEquipment(c, c.Param("operation")) })
	database := &repository.Database{DB: orm}
	native := NewPositionHandler(repository.NewPositionRepository(database), repository.NewJobRepository(database), repository.NewRequirementRepository(database), orm)
	router.PUT("/jobs/:id/positions/:posId", native.UpdatePosition)
	router.DELETE("/jobs/:id/positions/:posId", native.DeletePosition)
	return &externalCatalogFixture{db: db, router: router, t: t}, orm
}
func confirmAssignment(f *externalCatalogFixture, op string, in map[string]any) map[string]any {
	f.t.Helper()
	s, p := f.call(op, in, "")
	if s != 200 || p["ready_to_execute"] != true {
		f.t.Fatal("preview", op, s, p)
	}
	out := externalCatalogClone(in)
	for _, k := range []string{"expected_updated_at", "expected_position_updated_at", "expected_job_updated_at", "expected_context"} {
		if p[k] != nil {
			out[k] = p[k]
		}
	}
	out["confirmation_text"] = p["required_confirmation_text"]
	out["confirm_change"] = true
	return out
}
func createAssignment(f *externalCatalogFixture, equipment int) map[string]any {
	f.t.Helper()
	in := confirmAssignment(f, "create", map[string]any{"job_id": 1, "equipment_id": equipment, "quantity": 2, "days_used": 3})
	s, out := f.call("create", in, fmt.Sprintf("assignment-create-%d", equipment))
	if s != 201 {
		f.t.Fatal(s, out)
	}
	return map[string]any{"job_id": 1, "equipment_id": equipment, "position_id": out["position"].(map[string]any)["position_id"]}
}

func TestRentalAssignmentPartialUpdateSnapshotsAndAtomicReplay(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	f.exec(`UPDATE rental_equipment SET rental_price=99,customer_price=199 WHERE id=1`)
	patch := externalCatalogClone(base)
	patch["quantity"] = 3
	patch["days_used"] = 4
	patch["notes"] = "Retained assignment note"
	beforeAudit := f.count("audit_log")
	in := confirmAssignment(f, "update", patch)
	if f.count("audit_log") != beforeAudit {
		t.Fatal("preview wrote audit")
	}
	if s, _ := f.call("update", in, ""); s != 428 {
		t.Fatal("missing key accepted", s)
	}
	bad := externalCatalogClone(in)
	bad["confirmation_text"] = "UPDATE"
	if s, _ := f.call("update", bad, "assignment-bad-phrase"); s != 428 {
		t.Fatal(s)
	}
	s, out := f.call("update", in, "assignment-partial-once")
	if s != 200 || out["operation_status"] != "updated" {
		t.Fatal(s, out)
	}
	a := out["assignment"].(map[string]any)
	p := out["position"].(map[string]any)
	if a["total_cost"] != 148.2 || a["rental_unit_price"] != 12.35 || p["unit_price"] != float64(20) || p["quantity"] != float64(3) {
		t.Fatal("catalog repriced captured snapshots", out)
	}
	if f.count("job_rental_equipment") != 1 || f.count("job_positions") != 1 || f.count("audit_log") != beforeAudit+1 {
		t.Fatal("missing atomic records")
	}
	if s, replay := f.call("update", in, "assignment-partial-once"); s != 200 || stringMustJSON(replay) != stringMustJSON(out) {
		t.Fatal("original durable response changed", s, replay)
	}
	for _, financial := range []any{false, nil, "true", 1} {
		if s, _ := f.invoke("update", in, "assignment-partial-once", "cores:rental:update", financial); s != 403 {
			t.Fatal("financial replay bypass", s)
		}
	}
	f.exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ := f.call("update", in, "assignment-partial-once"); s != 403 {
		t.Fatal("revoked admin replay", s)
	}
	f.exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	if s, stale := f.call("update", in, "assignment-stale"); s != 200 || stale["ready_to_execute"] != false {
		t.Fatal("stale execution", s, stale)
	}
	pricePatch := externalCatalogClone(base)
	pricePatch["rental_unit_price"] = 0
	pricePatch["customer_price"] = 0
	s, zero := f.call("update", confirmAssignment(f, "update", pricePatch), "assignment-zero-prices")
	if s != 200 || zero["assignment"].(map[string]any)["total_cost"] != float64(0) || zero["effects"].(map[string]any)["revenue_after"] != float64(0) {
		t.Fatal("zero prices lost", s, zero)
	}
}

func TestRentalAssignmentArchiveFreezeRestoreAndNativeAudit(t *testing.T) {
	f, orm := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	archive := confirmAssignment(f, "archive", base)
	s, out := f.call("archive", archive, "assignment-archive-once")
	if s != 200 || out["operation_status"] != "archived" || out["effects"].(map[string]any)["revenue_after"] != float64(0) {
		t.Fatal("last archive did not clear revenue", s, out)
	}
	var captured string
	var cost float64
	f.db.QueryRow(`SELECT rental_unit_price::text,total_cost FROM job_rental_equipment WHERE job_id=1`).Scan(&captured, &cost)
	if cost != 74.10 {
		t.Fatal(cost)
	}
	f.exec(`UPDATE jobs SET multiply_by_days=false WHERE jobid=1;UPDATE job_rental_equipment SET updated_at=clock_timestamp() WHERE job_id=1`)
	var frozen float64
	f.db.QueryRow(`SELECT total_cost FROM job_rental_equipment WHERE job_id=1`).Scan(&frozen)
	if frozen != cost {
		t.Fatal("archived cost drifted", frozen, cost)
	}
	for _, q := range []string{`DELETE FROM job_rental_equipment WHERE job_id=1`, `UPDATE job_rental_equipment SET notes='overwrite' WHERE job_id=1`, `UPDATE job_positions SET unit_price=99 WHERE job_id=1`, `UPDATE job_rental_equipment SET position_id=NULL WHERE job_id=1`} {
		if _, err := f.db.Exec(q); err == nil {
			t.Fatal("retention bypass", q)
		}
	}
	f.exec(`UPDATE rental_equipment SET is_active=false WHERE id=1`)
	if s, p := f.call("restore", base, ""); s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), "restore_catalog_first") {
		t.Fatal("inactive catalog restore", s, p)
	}
	f.exec(`UPDATE rental_equipment SET is_active=true WHERE id=1`)
	restore := confirmAssignment(f, "restore", base)
	s, out = f.call("restore", restore, "assignment-restore-once")
	if s != 200 || out["assignment"].(map[string]any)["total_cost"] != 24.7 || out["assignment"].(map[string]any)["rental_unit_price_exact"] != captured {
		t.Fatal("restore repriced captured unit or ignored job setting", s, out)
	}
	if s, replay := f.call("archive", archive, "assignment-archive-once"); s != 200 || replay["operation_status"] != "archived" {
		t.Fatal("archive receipt replaced after restore", s, replay)
	}
	// Native changes commit their real before/after and ledger together.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", fmt.Sprintf("/jobs/1/positions/%.0f", base["position_id"]), strings.NewReader(`{"quantity":4}`))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var count int
	f.db.QueryRow(`SELECT count(*) FROM audit_log WHERE new_values->>'origin'='UI' AND new_values#>>'{after,quantity}'='4' AND old_values#>>'{assignment,quantity}'='2'`).Scan(&count)
	if count != 1 {
		t.Fatal("native real audit missing", count)
	}
	database := &repository.Database{DB: orm}
	repo := repository.NewRentalEquipmentRepository(database)
	if err := repo.AddRentalToJob(&models.JobRentalEquipment{JobID: 1, EquipmentID: 1, Quantity: 4, DaysUsed: 3}); err == nil {
		t.Fatal("legacy linked writer bypass")
	}
}

func TestRentalAssignmentValidationContextAndAuditRollback(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	for _, patch := range []map[string]any{{"quantity": 0}, {"days_used": 366}, {"notes": nil}, {"customer_price": 1.001}, {"rental_unit_price": -1}, {"equipment_id": 0}, {"table": "job_positions"}, {"tax_rate": 101}, {"tax_rate": 19.123}, {"follow_day_factor": 100}} {
		in := externalCatalogClone(base)
		for k, v := range patch {
			in[k] = v
		}
		if s, _ := f.call("update", in, ""); s != 400 {
			t.Fatal("invalid input", in, s)
		}
	}
	extra := externalCatalogClone(base)
	extra["notes"] = "lifecycle overwrite"
	if s, _ := f.call("archive", extra, ""); s != 400 {
		t.Fatal("archive business change", s)
	}
	patch := externalCatalogClone(base)
	patch["notes"] = "New note"
	in := confirmAssignment(f, "update", patch)
	f.exec(`INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	if s, p := f.call("update", in, "assignment-editor-stale"); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("editor context ignored", s, p)
	}
	f.exec(`DELETE FROM job_edit_sessions`)
	in = confirmAssignment(f, "update", patch)
	f.exec(`CREATE FUNCTION fail_assignment_audit() RETURNS trigger AS $$BEGIN RAISE EXCEPTION 'synthetic audit failure';END;$$ LANGUAGE plpgsql;CREATE TRIGGER fail_assignment_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_assignment_audit()`)
	before := f.count("rental_mcp_mutation_receipts")
	if s, _ := f.call("update", in, "assignment-audit-rollback"); s != 500 {
		t.Fatal("audit failure", s)
	}
	var notes string
	f.db.QueryRow(`SELECT notes FROM job_rental_equipment WHERE job_id=1`).Scan(&notes)
	if notes != "" || f.count("rental_mcp_mutation_receipts") != before {
		t.Fatal("audit left partial data", notes)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", fmt.Sprintf("/jobs/1/positions/%.0f", base["position_id"]), strings.NewReader(`{"quantity":4}`))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	if w.Code != 500 {
		t.Fatal("native audit failure", w.Code, w.Body.String())
	}
	var qty int
	if err := f.db.QueryRow(`SELECT quantity FROM job_rental_equipment WHERE job_id=1`).Scan(&qty); err != nil || qty != 2 {
		t.Fatal("native audit left partial cost/quantity", qty, err)
	}
	f.exec(`DROP TRIGGER fail_assignment_audit ON audit_log`)
	if s, _ := f.call("update", in, "assignment-audit-rollback"); s != 200 {
		t.Fatal("same-key retry failed", s)
	}
}

func TestRentalAssignmentPreciseLegacyPriceAndSchemaRestart(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	f.exec(`UPDATE job_rental_equipment SET rental_unit_price=1.234567890123456789 WHERE job_id=1`)
	patch := externalCatalogClone(base)
	patch["days_used"] = 7
	s, out := f.call("update", confirmAssignment(f, "update", patch), "assignment-precise-price")
	if s != 200 || out["assignment"].(map[string]any)["rental_unit_price_exact"] != "1.234567890123456789" {
		t.Fatal("captured numeric precision lost", s, out)
	}
	for _, ensure := range []func(*sql.DB) error{schema.EnsureRentalPositionLifecycle, schema.EnsureRentalAssignmentLifecycle, schema.EnsureRentalExternalCatalogLifecycle, schema.EnsureRentalAssignmentLifecycle} {
		if err := ensure(f.db); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile("../../migrations/053_rental_assignment_lifecycle.sql")
	if err != nil || strings.TrimSpace(string(raw)) != strings.TrimSpace(schema.RentalAssignmentLifecycleSQL) {
		t.Fatal("migration/startup mismatch", err)
	}
	s, out = f.call("archive", confirmAssignment(f, "archive", base), "assignment-after-restart-archive")
	if s != 200 {
		t.Fatal(s, out)
	}
	s, out = f.call("restore", confirmAssignment(f, "restore", base), "assignment-after-restart-restore")
	if s != 200 {
		t.Fatal(s, out)
	}
}

func TestRentalAssignmentNativeDelegationBlockedBeforeDatabase(t *testing.T) {
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("native-assignment-", 3))
	gin.SetMode(gin.TestMode)
	h := &PositionHandler{}
	job := &JobHandler{}
	pdf := &PDFHandler{}
	routes := []gin.HandlerFunc{h.CreatePosition, h.UpdatePosition, h.DeletePosition, h.ReorderPositions, h.AssignDevice, h.RemoveDevice, h.UpdatePriceSettings, job.CreateJob, job.UpdateJob, pdf.FinalizeExtraction}
	for i, handler := range routes {
		for _, scope := range []any{"cores:rental:update", nil, 1, []string{"cores:rental:update"}} {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": scope, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.POST("/write", handler)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/write", nil)
			req.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
			router.ServeHTTP(w, req)
			if w.Code != 403 {
				t.Fatal("native bypass", i, scope, w.Code)
			}
		}
	}
}

// Archive versus quantity update: one winner, no split position/cost snapshots.
func TestRentalAssignmentArchiveNativeQuantityRaces(t *testing.T) {
	for _, archiveFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(archiveFirst), func(t *testing.T) {
			f, _ := newAssignmentFixture(t)
			base := createAssignment(f, 1)
			id := int64(base["position_id"].(float64))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			first, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer first.Rollback()
			second, err := f.db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer second.Close()
			var pid int
			second.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid)
			archive := fmt.Sprintf(`UPDATE job_positions SET deleted_at=clock_timestamp() WHERE position_id=%d`, id)
			update := fmt.Sprintf(`UPDATE job_positions SET quantity=4 WHERE position_id=%d`, id)
			holding, waiting := update, archive
			if archiveFirst {
				holding, waiting = archive, update
			}
			if _, err = first.Exec(holding); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, e := second.ExecContext(ctx, waiting); done <- e }()
			waitExternalCatalogBlocked(t, f.db, pid)
			if err = first.Commit(); err != nil {
				t.Fatal(err)
			}
			err = <-done
			if archiveFirst {
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23514" {
					t.Fatal("archived position updated", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var qty, ledgerQty int
			var cost float64
			if err = f.db.QueryRow(`SELECT p.quantity::int,c.quantity,c.total_cost FROM job_positions p JOIN job_rental_equipment c ON c.position_id=p.position_id WHERE p.position_id=$1`, id).Scan(&qty, &ledgerQty, &cost); err != nil {
				t.Fatal(err)
			}
			if qty != ledgerQty || cost != roundAnalyticsMoney(float64(qty)*3*12.35) {
				t.Fatal("split position/ledger", qty, ledgerQty, cost)
			}
		})
	}
}

func TestRentalAssignmentNativeJobLockAndRepairSerialize(t *testing.T) {
	f, orm := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	native, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer native.Rollback()
	if _, err = native.Exec(`SELECT jobid FROM jobs WHERE jobid=1 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if s, _ := f.call("archive", base, ""); s != 409 {
		t.Fatal("busy native job not rejected", s)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("owner waited while holding child locks")
	}
	if _, err = native.Exec(`UPDATE job_rental_equipment SET notes='Native completion' WHERE job_id=1`); err != nil {
		t.Fatal("owner left reverse lock dependency", err)
	}
	if err = native.Commit(); err != nil {
		t.Fatal(err)
	}
	// Hold a real explicit legacy repair transaction. A native legacy writer must
	// not pass an old unlinked lookup and subsequently overwrite the linked row.
	f.exec(`INSERT INTO job_rental_equipment(job_id,equipment_id,quantity,days_used,total_cost,notes) VALUES(1,2,2,3,45,'Legacy snapshot')`)
	repair, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer repair.Rollback()
	if _, err = repair.Exec(`SELECT jobid FROM jobs WHERE jobid=1 FOR UPDATE;INSERT INTO job_positions(job_id,position_type,rental_equipment_id,description,quantity,unit_price) VALUES(1,'rental',2,'Repair',2,15);UPDATE job_rental_equipment SET position_id=(SELECT position_id FROM job_positions WHERE job_id=1 AND rental_equipment_id=2),rental_unit_price=7.5 WHERE job_id=1 AND equipment_id=2`); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRentalEquipmentRepository(&repository.Database{DB: orm})
	legacy := &models.JobRentalEquipment{JobID: 1, EquipmentID: 2, Quantity: 2, DaysUsed: 9, Notes: "Must not overwrite"}
	if err = repo.AddRentalToJob(legacy); err == nil {
		t.Fatal("legacy writer crossed pending repair")
	}
	if err = repair.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = repo.AddRentalToJob(legacy); err == nil {
		t.Fatal("legacy writer crossed completed repair")
	}
	var notes string
	f.db.QueryRow(`SELECT notes FROM job_rental_equipment WHERE job_id=1 AND equipment_id=2`).Scan(&notes)
	if notes != "Legacy snapshot" {
		t.Fatal("legacy overwrite after repair", notes)
	}
}

func TestRentalAssignmentMaximumUnicodeDescription(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	base["description"] = strings.Repeat("🔧", 10000)
	s, out := f.call("update", confirmAssignment(f, "update", base), "assignment-long-unicode")
	if s != 200 || out["position"].(map[string]any)["description"] != base["description"] {
		t.Fatal("valid multi-byte description rejected", s, out)
	}
}

func TestRentalAssignmentNativeRequestCannotOverwriteAfterRestore(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	base := createAssignment(f, 1)
	id := int64(base["position_id"].(float64))
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var firstPID int
	if err = tx.QueryRow(`SELECT pg_backend_pid()`).Scan(&firstPID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`SELECT jobid FROM jobs WHERE jobid=1 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	finished := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", fmt.Sprintf("/jobs/1/positions/%d", id), strings.NewReader(`{"quantity":4}`))
		req.Header.Set("Content-Type", "application/json")
		f.router.ServeHTTP(w, req)
		finished <- w.Code
	}()
	deadline := time.Now().Add(3 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		if err = f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, firstPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("native PUT did not reach parent lock after initial read")
	}
	if _, err = tx.Exec(`UPDATE job_positions SET unit_price=27 WHERE position_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE job_positions SET deleted_at=clock_timestamp() WHERE position_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE job_positions SET deleted_at=NULL WHERE position_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-finished:
		if status != 409 {
			t.Fatal("native stale request accepted", status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("native PUT did not finish")
	}
	var price float64
	var qty int
	if err = f.db.QueryRow(`SELECT unit_price,quantity::int FROM job_positions WHERE position_id=$1`, id).Scan(&price, &qty); err != nil || price != 27 || qty != 2 {
		t.Fatal("stale native fields overwrote restored position", price, qty, err)
	}
}

func TestRentalAssignmentCommercialPricingMatchesConfirmedEffects(t *testing.T) {
	for _, inclusive := range []bool{false, true} {
		t.Run(fmt.Sprint(inclusive), func(t *testing.T) {
			f, _ := newAssignmentFixture(t)
			base := createAssignment(f, 1)
			f.exec(`UPDATE jobs SET prices_include_tax=$1,discount=10,discount_type='percent' WHERE jobid=1`, inclusive)
			patch := externalCatalogClone(base)
			patch["quantity"] = 3
			patch["days_used"] = 5
			patch["customer_price"] = 10
			patch["follow_day_factor"] = 0.5
			patch["discount_percent"] = 10
			patch["discount_amount"] = 2
			patch["tax_rate"] = 19
			s, out := f.call("update", confirmAssignment(f, "update", patch), "assignment-commercial-pricing")
			if s != 200 {
				t.Fatal(s, out)
			}
			effects := out["effects"].(map[string]any)
			var revenue, final float64
			if err := f.db.QueryRow(`SELECT revenue,final_revenue FROM jobs WHERE jobid=1`).Scan(&revenue, &final); err != nil {
				t.Fatal(err)
			}
			// Existing Core contract uses end-start: October 1 to 4 is 3 event days.
			wantRevenue, wantFinal, wantNet := 61.88, 55.69, 46.80
			if inclusive {
				wantRevenue, wantFinal, wantNet = 52.00, 46.80, 39.33
			}
			if revenue != wantRevenue || final != wantFinal || effects["revenue_after"] != wantRevenue || effects["final_revenue_after"] != wantFinal || effects["customer_sales_net_after"] != wantNet || effects["supplier_cost_after"] != 185.25 {
				t.Fatal("confirmed pricing differs from stored job or golden", inclusive, revenue, final, effects)
			}
		})
	}
}

func TestRentalAssignmentCreateRepairBusyTableReturnsConflict(t *testing.T) {
	f, _ := newAssignmentFixture(t)
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`LOCK TABLE job_positions IN ROW EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	s, _ := f.call("create", map[string]any{"job_id": 1, "equipment_id": 1, "quantity": 1, "days_used": 1}, "")
	if s != 409 || time.Since(started) > 2*time.Second {
		t.Fatal("create/repair table collision did not return prompt409", s)
	}
}
