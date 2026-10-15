package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/schema"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type externalCatalogFixture struct {
	db     *sql.DB
	router *gin.Engine
	t      *testing.T
}

func newExternalCatalogFixture(t *testing.T) *externalCatalogFixture {
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
	ns := "external_catalog_" + hex.EncodeToString(sum[:6])
	q := u.Query()
	q.Set("search_path", ns)
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { db.Exec("DROP SCHEMA " + ns + " CASCADE"); db.Close() })
	if _, err = db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE;CREATE SCHEMA " + ns); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/external_catalog_lifecycle.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(schema.RentalExternalCatalogLifecycleSQL); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("external-lifecycle-", 3))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	h := &RentalExternalEquipmentMCP{db: db}
	r.POST("/create", h.Create)
	r.POST("/:operation", h.Change)
	return &externalCatalogFixture{db: db, router: r, t: t}
}
func (f *externalCatalogFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		f.t.Fatal(err)
	}
}
func (f *externalCatalogFixture) count(table string) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}
func (f *externalCatalogFixture) invoke(op string, in map[string]any, key, scope string, financial any) (int, map[string]any) {
	f.t.Helper()
	raw, _ := json.Marshal(in)
	req := httptest.NewRequest("POST", "/"+op, bytes.NewReader(raw))
	req.Header.Set("X-Cores-Origin", "MCP/AI")
	req.Header.Set("Idempotency-Key", key)
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": scope, "mcp_financial": financial, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
	if err != nil {
		f.t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "cores_token", Value: signed})
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	out := map[string]any{}
	if err = json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		f.t.Fatal(w.Body.String())
	}
	return w.Code, out
}
func (f *externalCatalogFixture) call(op string, in map[string]any, key string) (int, map[string]any) {
	action := op
	if op == "restore" {
		action = "archive"
	}
	return f.invoke(op, in, key, "cores:rental:"+action, true)
}
func (f *externalCatalogFixture) confirmed(op string, in map[string]any) map[string]any {
	f.t.Helper()
	status, p := f.call(op, in, "")
	if status != 200 || p["ready_to_execute"] != true {
		f.t.Fatalf("preview %s: %d %v", op, status, p)
	}
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	out["expected_updated_at"] = p["expected_updated_at"]
	out["expected_context"] = p["expected_context"]
	out["confirmation_text"] = p["required_confirmation_text"]
	out["confirm_change"] = true
	return out
}
func externalCatalogClone(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func TestExternalCatalogPartialUpdateVersionsSnapshotsAndRights(t *testing.T) {
	f := newExternalCatalogFixture(t)
	f.exec(`INSERT INTO job_positions(job_id,rental_equipment_id,unit_price) VALUES(1,1,25.50);INSERT INTO job_rental_equipment(job_id,equipment_id,position_id,quantity,days_used,rental_unit_price,total_cost) VALUES(1,1,1,2,3,12.35,74.10)`)
	patch := map[string]any{"equipment_id": 1, "rental_price": 13.25, "customer_price": 0, "description": ""}
	s, p := f.call("update", patch, "")
	if s != 200 || p["ready_to_execute"] != true {
		t.Fatal(s, p)
	}
	if f.count("audit_log") != 0 || f.count("rental_mcp_mutation_receipts") != 0 {
		t.Fatal("preview wrote data")
	}
	current := p["current"].(map[string]any)
	draft := p["draft"].(map[string]any)
	if current["notes"] != draft["notes"] || draft["customer_price"] != float64(0) || draft["supplier_id"] != float64(1) || len(p["custom_fields"].([]any)) != 1 {
		t.Fatal("partial draft lost fields", p)
	}
	in := f.confirmed("update", patch)
	if s, _ := f.call("update", in, ""); s != 428 {
		t.Fatal("missing durable key accepted", s)
	}
	bad := externalCatalogClone(in)
	bad["confirmation_text"] = "UPDATE"
	if s, _ := f.call("update", bad, "bad-catalog-phrase"); s != 428 {
		t.Fatal("unbound phrase accepted", s)
	}
	s, out := f.call("update", in, "catalog-update-once")
	if s != 200 || out["operation_status"] != "updated" {
		t.Fatal(s, out)
	}
	record := out["equipment"].(map[string]any)
	if record["updated_at"] == current["updated_at"] || record["rental_price"] != 13.25 || record["notes"] != current["notes"] {
		t.Fatal("mutation/version lost", record)
	}
	var sales, cost float64
	if err := f.db.QueryRow(`SELECT p.unit_price,r.total_cost FROM job_positions p JOIN job_rental_equipment r ON r.position_id=p.position_id WHERE p.position_id=1`).Scan(&sales, &cost); err != nil || sales != 25.50 || cost != 74.10 {
		t.Fatal("catalog repriced existing job", sales, cost, err)
	}
	if f.count("audit_log") != 1 || f.count("rental_mcp_mutation_receipts") != 1 {
		t.Fatal("audit/receipt missing")
	}
	f.exec(`UPDATE rental_equipment SET category='Native category',updated_at='2000-01-01' WHERE id=1`)
	if s, replay := f.call("update", in, "catalog-update-once"); s != 200 || stringMustJSON(replay) != stringMustJSON(out) {
		t.Fatal("durable original response changed", s, replay)
	}
	for _, fin := range []any{false, nil, "true", 1} {
		if s, _ := f.invoke("update", in, "catalog-update-once", "cores:rental:update", fin); s != 403 {
			t.Fatal("financial replay bypass", fin, s)
		}
	}
	f.exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ := f.call("update", in, "catalog-update-once"); s != 403 {
		t.Fatal("revoked admin replay", s)
	}
	f.exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	if s, stale := f.call("update", in, "catalog-stale-key"); s != 200 || stale["ready_to_execute"] != false || !strings.Contains(stringMustJSON(stale["required_fields"]), "expected_") {
		t.Fatal("stale preview accepted", s, stale)
	}
	changed := externalCatalogClone(in)
	changed["rental_price"] = 15.25
	if s, _ := f.call("update", changed, "catalog-update-once"); s != 409 {
		t.Fatal("key payload conflict accepted", s)
	}
}

func TestExternalCatalogArchiveRestoreDependenciesAndCustomContext(t *testing.T) {
	f := newExternalCatalogFixture(t)
	f.exec(`INSERT INTO job_rental_equipment(job_id,equipment_id,rental_unit_price,total_cost) VALUES(1,1,12.35,12.35)`)
	for _, op := range []string{"archive", "update"} {
		in := map[string]any{"equipment_id": 1}
		if op == "update" {
			in["supplier_id"] = 0
		}
		s, p := f.call(op, in, "")
		if s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), "active_jobs") {
			t.Fatal("unlinked legacy reference ignored", op, s, p)
		}
	}
	f.exec(`UPDATE jobs SET statusid=2 WHERE jobid=1;INSERT INTO job_positions(job_id,rental_equipment_id,unit_price) VALUES(3,1,25.50)`)
	if s, p := f.call("archive", map[string]any{"equipment_id": 1}, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("position without ledger ignored", s, p)
	}
	f.exec(`UPDATE jobs SET statusid=3 WHERE jobid=3`)
	in := f.confirmed("archive", map[string]any{"equipment_id": 1})
	f.exec(`UPDATE rental_equipment_field_values SET value='IP65' WHERE equipment_id=1`)
	if s, p := f.call("archive", in, "catalog-custom-stale"); s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), "expected_context") {
		t.Fatal("custom context change ignored", s, p)
	}
	in = f.confirmed("archive", map[string]any{"equipment_id": 1})
	s, archived := f.call("archive", in, "catalog-archive-once")
	if s != 200 || archived["operation_status"] != "archived" {
		t.Fatal(s, archived)
	}
	if s, p := f.call("update", map[string]any{"equipment_id": 1, "notes": "overwrite"}, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("archived edit accepted", s, p)
	}
	f.exec(`INSERT INTO jobs(jobid,job_code,statusid) VALUES(4,'SYNTHETIC-NEW',1)`)
	for _, q := range []string{`DELETE FROM rental_equipment WHERE id=1`, `UPDATE rental_equipment SET notes='overwrite' WHERE id=1`, `UPDATE jobs SET statusid=1 WHERE jobid=1`, `INSERT INTO job_positions(job_id,rental_equipment_id) VALUES(4,1)`, `INSERT INTO job_rental_equipment(job_id,equipment_id) VALUES(4,1)`} {
		if _, err := f.db.Exec(q); err == nil {
			t.Fatal("native lifecycle guard bypass", q)
		}
	}

	f.exec(`UPDATE customers SET is_archived=true WHERE customerid=1`)
	if s, p := f.call("restore", map[string]any{"equipment_id": 1}, ""); s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), "active_supplier_id") {
		t.Fatal("inactive supplier restore", s, p)
	}
	f.exec(`UPDATE customers SET is_archived=false WHERE customerid=1`)
	restore := f.confirmed("restore", map[string]any{"equipment_id": 1})
	s, out := f.call("restore", restore, "catalog-restore-once")
	if s != 200 || out["operation_status"] != "restored" {
		t.Fatal(s, out)
	}
	var custom string
	f.db.QueryRow(`SELECT value FROM rental_equipment_field_values WHERE equipment_id=1`).Scan(&custom)
	if custom != "IP65" || f.count("job_rental_equipment") != 1 || f.count("job_positions") != 1 {
		t.Fatal("lifecycle lost history/custom values")
	}
	if s, replay := f.call("archive", in, "catalog-archive-once"); s != 200 || stringMustJSON(replay) != stringMustJSON(archived) {
		t.Fatal("original archive receipt changed after restore", s, replay)
	}
}

func TestExternalCatalogValidationDuplicatesAndAuditRollback(t *testing.T) {
	f := newExternalCatalogFixture(t)
	for _, patch := range []map[string]any{{"equipment_id": 1, "notes": nil}, {"equipment_id": 1, "rental_price": 1.001}, {"equipment_id": 1, "customer_price": -1}, {"equipment_id": 1, "supplier_id": -1}, {"equipment_id": 1, "job_id": 1}, {"equipment_id": 1, "name": strings.Repeat("x", 256)}} {
		if s, _ := f.call("update", patch, ""); s != 400 {
			t.Fatal("invalid input accepted", patch, s)
		}
	}
	if s, _ := f.call("archive", map[string]any{"equipment_id": 1, "rental_price": 0}, ""); s != 400 {
		t.Fatal("archive changed price", s)
	}
	f.exec(`INSERT INTO rental_equipment(name,supplier,category,rental_price,customer_price) VALUES('Another Rental','Synthetic Supplier','Audio',1,2)`)
	if s, p := f.call("update", map[string]any{"equipment_id": 1, "name": "Another Rental", "allow_similar": true}, ""); s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), "reuse_existing_equipment") {
		t.Fatal("exact duplicate overridden", s, p)
	}
	if _, err := f.db.Exec(`INSERT INTO rental_equipment(name,supplier) VALUES(' synthetic rental ',' synthetic supplier ')`); err == nil {
		t.Fatal("native duplicate bypass")
	}
	in := f.confirmed("update", map[string]any{"equipment_id": 1, "notes": "New business note"})
	f.exec(`CREATE FUNCTION fail_catalog_audit() RETURNS trigger AS $$BEGIN RAISE EXCEPTION 'synthetic audit failure';END;$$ LANGUAGE plpgsql;CREATE TRIGGER fail_catalog_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_catalog_audit()`)
	if s, _ := f.call("update", in, "catalog-audit-rollback"); s != 500 {
		t.Fatal("audit failure did not abort", s)
	}
	var notes string
	f.db.QueryRow(`SELECT notes FROM rental_equipment WHERE id=1`).Scan(&notes)
	if notes != "Private catalog note" || f.count("audit_log") != 0 || f.count("rental_mcp_mutation_receipts") != 0 {
		t.Fatal("audit failure left partial data")
	}
}

func waitExternalCatalogBlocked(t *testing.T, db *sql.DB, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		if err := db.QueryRow(`SELECT cardinality(pg_blocking_pids($1))>0`, pid).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("second real connection did not reach the intended catalog lock")
}

func TestExternalCatalogNativeReferenceAndReopenRaces(t *testing.T) {
	for _, kind := range []string{"assignment", "position", "reopen"} {
		for _, archiveFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/archive_first_%v", kind, archiveFirst), func(t *testing.T) {
				f := newExternalCatalogFixture(t)
				if kind == "reopen" {
					f.exec(`INSERT INTO job_rental_equipment(job_id,equipment_id,total_cost) VALUES(2,1,12.35)`)
				}
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
				var pid1, pid2 int
				first.QueryRow(`SELECT pg_backend_pid()`).Scan(&pid1)
				second.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid2)
				if pid1 == pid2 {
					t.Fatal("race reused connection")
				}
				reference := `INSERT INTO job_rental_equipment(job_id,equipment_id,total_cost) VALUES(1,1,12.35)`
				if kind == "position" {
					reference = `INSERT INTO job_positions(job_id,rental_equipment_id,unit_price) VALUES(1,1,25.50)`
				}
				if kind == "reopen" {
					reference = `UPDATE jobs SET statusid=1 WHERE jobid=2`
				}
				archive := `UPDATE rental_equipment SET is_active=false WHERE id=1`
				holding, waiting := reference, archive
				if archiveFirst {
					holding, waiting = archive, reference
				}
				if _, err = first.Exec(holding); err != nil {
					t.Fatal(err)
				}
				finished := make(chan error, 1)
				go func() { _, e := second.ExecContext(ctx, waiting); finished <- e }()
				waitExternalCatalogBlocked(t, f.db, pid2)
				if err = first.Commit(); err != nil {
					t.Fatal(err)
				}
				err = <-finished
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23514" {
					t.Fatal("stale concurrent operation committed", err)
				}
				var active bool
				f.db.QueryRow(`SELECT is_active FROM rental_equipment WHERE id=1`).Scan(&active)
				if active == archiveFirst {
					t.Fatal("catalog state inconsistent")
				}
			})
		}
	}
}

// A closed job with no references must serialize reopening with its first new
// reference. Checking the old MVCC status alone allowed both operations through.
func TestExternalCatalogFirstReferenceReopenRaces(t *testing.T) {
	for _, kind := range []string{"assignment", "position"} {
		for _, reopenFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reopen_first_%v", kind, reopenFirst), func(t *testing.T) {
				f := newExternalCatalogFixture(t)
				f.exec(`UPDATE rental_equipment SET is_active=false WHERE id=1`)
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
				if err = second.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
					t.Fatal(err)
				}
				reference := `INSERT INTO job_rental_equipment(job_id,equipment_id,total_cost) VALUES(2,1,12.35)`
				if kind == "position" {
					reference = `INSERT INTO job_positions(job_id,rental_equipment_id,unit_price) VALUES(2,1,25.50)`
				}
				holding, waiting := reference, `UPDATE jobs SET statusid=1 WHERE jobid=2`
				if reopenFirst {
					holding, waiting = waiting, holding
				}
				if _, err = first.ExecContext(ctx, holding); err != nil {
					t.Fatal(err)
				}
				finished := make(chan error, 1)
				go func() { _, e := second.ExecContext(ctx, waiting); finished <- e }()
				waitExternalCatalogBlocked(t, f.db, pid)
				if err = first.Commit(); err != nil {
					t.Fatal(err)
				}
				err = <-finished
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23514" {
					t.Fatal("first reference and reopen both committed", err)
				}
			})
		}
	}
}

func TestExternalCatalogOwnerDoesNotWaitOnNativeChildTable(t *testing.T) {
	f := newExternalCatalogFixture(t)
	// Install the actual existing job lifecycle, including child guards and the
	// AFTER trigger that updates jobs while the child table is RowExclusive.
	if _, err := f.db.Exec(schema.RentalJobLifecycleSQL); err != nil {
		t.Fatal(err)
	}
	f.exec(`INSERT INTO job_rental_equipment(job_id,equipment_id,total_cost) VALUES(1,1,12.35)`)
	native, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer native.Rollback()
	if _, err = native.Exec(`LOCK TABLE job_positions,job_rental_equipment IN ROW EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		s, _ := f.call("update", map[string]any{"equipment_id": 1, "notes": "New note"}, "")
		result <- s
	}()
	select {
	case s := <-result:
		if s != 200 {
			t.Fatal("owner preview failed", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("owner waits for native child table while retaining job locks")
	}
	if _, err = native.Exec(`UPDATE job_rental_equipment SET quantity=2 WHERE job_id=1`); err != nil {
		t.Fatal("real native child trigger failed", err)
	}
	if err = native.Commit(); err != nil {
		t.Fatal(err)
	}
	// Once native work holds the job version row, the owner must fail promptly
	// with a retryable conflict instead of waiting while retaining catalog locks.
	native, err = f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer native.Rollback()
	if _, err = native.Exec(`UPDATE job_rental_equipment SET quantity=3 WHERE job_id=1`); err != nil {
		t.Fatal(err)
	}
	go func() {
		s, _ := f.call("update", map[string]any{"equipment_id": 1, "notes": "New note"}, "")
		result <- s
	}()
	select {
	case s := <-result:
		if s != 409 {
			t.Fatal("busy job did not return conflict", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("owner waited on native job version lock")
	}
	if err = native.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestExternalCatalogSchemaSourcesAndReapplication(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/052_rental_external_catalog_lifecycle.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != strings.TrimSpace(schema.RentalExternalCatalogLifecycleSQL) {
		t.Fatal("standalone migration differs from startup SQL")
	}
	f := newExternalCatalogFixture(t)
	if err = schema.EnsureRentalExternalCatalogLifecycle(f.db); err != nil {
		t.Fatal(err)
	}
	if err = schema.EnsureRentalExternalCatalogLifecycle(f.db); err != nil {
		t.Fatal(err)
	}
	if f.count("rental_equipment") != 1 || f.count("rental_equipment_field_values") != 1 {
		t.Fatal("reapplication changed catalog")
	}
}
