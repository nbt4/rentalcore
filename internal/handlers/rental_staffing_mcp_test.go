package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func newStaffingFixture(t *testing.T) *externalCatalogFixture {
	t.Helper()
	f, orm := newAssignmentFixture(t)
	for i := 0; i < 2; i++ {
		if err := schema.EnsureRentalStaffingLifecycle(f.db); err != nil {
			t.Fatal(err)
		}
	}
	f.exec(`INSERT INTO employees(id,first_name,last_name,email,street,iban,notes) VALUES(1,'Ada','Staffing','private@example.invalid','Private address','Private bank','Private note'),(2,'Ben','Staffing',NULL,NULL,NULL,NULL);INSERT INTO skills(id,name,category) VALUES(1,'Audio','Technik');INSERT INTO employee_skills VALUES(1,1)`)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	h := &RentalStaffingMCP{db: f.db}
	router.POST("/:operation", h.Change)
	native := &JobHandler{jobRepo: repository.NewJobRepository(&repository.Database{DB: orm}), jobEmployeeRepo: repository.NewJobEmployeeRepository(orm)}
	router.POST("/jobs/:id/employees", native.AssignEmployee)
	router.DELETE("/jobs/:id/employees/:employeeId", native.RemoveEmployee)
	f.router = router
	return f
}
func confirmStaffing(f *externalCatalogFixture, op string, in map[string]any) map[string]any {
	f.t.Helper()
	s, p := f.call(op, in, "")
	if s != 200 || p["ready_to_execute"] != true {
		f.t.Fatal(op, s, p)
	}
	out := externalCatalogClone(in)
	for _, k := range []string{"expected_updated_at", "expected_job_updated_at", "expected_context"} {
		if p[k] != nil {
			out[k] = p[k]
		}
	}
	out["confirm_change"] = true
	out["confirmation_text"] = p["required_confirmation_text"]
	return out
}
func createStaffing(f *externalCatalogFixture, employee int) map[string]any {
	f.t.Helper()
	in := confirmStaffing(f, "create", map[string]any{"job_id": 1, "employee_id": employee, "role": "Ton"})
	if s, out := f.call("create", in, "staff-create-initial"); s != 201 {
		f.t.Fatal(s, out)
	}
	return map[string]any{"job_id": 1, "employee_id": employee}
}
func TestRentalStaffingLifecycleVersionsRetainedRolesReplayAndPrivacy(t *testing.T) {
	f := newStaffingFixture(t)
	base := createStaffing(f, 1)
	if s, p := f.call("create", base, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("duplicate", s, p)
	}
	patch := externalCatalogClone(base)
	patch["role"] = "Licht"
	for _, op := range []string{"update", "archive", "restore"} {
		input := base
		if op == "update" {
			input = patch
		}
		in := confirmStaffing(f, op, input)
		before := f.count("audit_log")
		if strings.Contains(prettyStaffing(in), "Private") {
			t.Fatal("private preview")
		}
		if s, _ := f.call(op, in, ""); s != 428 {
			t.Fatal("missing key", s)
		}
		wrong := externalCatalogClone(in)
		wrong["confirmation_text"] = "YES"
		if s, _ := f.call(op, wrong, "staff-wrong-phrase-"+op); s != 428 {
			t.Fatal(s)
		}
		s, out := f.invoke(op, in, "staff-lifecycle-"+op, "cores:rental:"+map[string]string{"update": "update", "archive": "archive", "restore": "archive"}[op], false)
		if s != 200 {
			t.Fatal(s, out)
		}
		after := out["assignment"].(map[string]any)
		if after["role"] != "Licht" || after["is_archived"] != (op == "archive") || after["updated_at"] == in["expected_updated_at"] {
			t.Fatal(after)
		}
		s, replay := f.call(op, in, "staff-lifecycle-"+op)
		if s != 200 || prettyStaffing(out) != prettyStaffing(replay) || f.count("audit_log") != before+1 {
			t.Fatal("replay", s, replay)
		}
		changed := externalCatalogClone(in)
		changed["confirmation_text"] = "changed"
		if s, _ = f.call(op, changed, "staff-lifecycle-"+op); s != 409 {
			t.Fatal("key mismatch", s)
		}
		f.exec(`UPDATE users SET is_admin=false WHERE userid=1`)
		if s, _ = f.call(op, in, "staff-lifecycle-"+op); s != 403 {
			t.Fatal("replay current rights", s)
		}
		f.exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	}
	var private bool
	if e := f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM audit_log WHERE entity_type='rental_staffing' AND (new_values::text LIKE '%Private%' OR old_values::text LIKE '%Private%' OR new_values::text LIKE '%email%'))`).Scan(&private); e != nil || private {
		t.Fatal("audit privacy", e)
	}
	var assignments int
	f.db.QueryRow(`SELECT count(*) FROM job_employees`).Scan(&assignments)
	if assignments != 1 {
		t.Fatal("retained identity", assignments)
	}
}
func prettyStaffing(v any) string { b, _ := json.Marshal(v); return string(b) }
func TestRentalStaffingStaleVersionsConflictEditorsAndInactive(t *testing.T) {
	f := newStaffingFixture(t)
	base := map[string]any{"job_id": 1, "employee_id": 1, "role": "Ton"}
	in := confirmStaffing(f, "create", base)
	f.exec(`UPDATE employees SET first_name='Changed' WHERE id=1`)
	if s, p := f.call("create", in, "staff-stale-employee"); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("stale employee", s, p)
	}
	f.exec(`INSERT INTO jobs(jobid,job_code,customerid,statusid,startdate,enddate) VALUES(2,'OVERLAP',1,1,'2026-10-03','2026-10-06');INSERT INTO job_employees(job_id,employee_id,role) VALUES(2,1,'Other')`)
	if s, p := f.call("create", base, ""); s != 200 || p["ready_to_execute"] != false || len(p["schedule_conflicts"].([]any)) != 1 {
		t.Fatal("overlap", s, p)
	}
	base["allow_overlap"] = true
	in = confirmStaffing(f, "create", base)
	f.exec(`UPDATE jobs SET enddate='2026-10-07' WHERE jobid=2`)
	if s, p := f.call("create", in, "staff-stale-conflict"); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("changed conflict", s, p)
	}
	in = confirmStaffing(f, "create", base)
	if s, p := f.call("create", in, "staff-allow-overlap"); s != 201 {
		t.Fatal(s, p)
	}
	patch := map[string]any{"job_id": 1, "employee_id": 1, "role": "Updated"}
	in = confirmStaffing(f, "update", patch)
	f.exec(`UPDATE job_employees SET role='Native' WHERE job_id=1 AND employee_id=1`)
	if s, p := f.call("update", in, "staff-stale-native"); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("stale native", s, p)
	}
	f.exec(`INSERT INTO job_edit_sessions(job_id,user_id) VALUES(1,2)`)
	if s, p := f.call("update", patch, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("editor", s, p)
	}
	f.exec(`DELETE FROM job_edit_sessions;UPDATE employees SET is_active=false WHERE id=1`)
	if s, p := f.call("update", patch, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("inactive update", s, p)
	}
	a := confirmStaffing(f, "archive", map[string]any{"job_id": 1, "employee_id": 1})
	if s, p := f.call("archive", a, "staff-inactive-archive"); s != 200 {
		t.Fatal(s, p)
	}
	if s, p := f.call("restore", map[string]any{"job_id": 1, "employee_id": 1, "allow_overlap": true}, ""); s != 200 || p["ready_to_execute"] != false {
		t.Fatal("inactive restore", s, p)
	}
}
func TestRentalStaffingAtomicAuditHistoryRollbackAndDatabaseGuards(t *testing.T) {
	f := newStaffingFixture(t)
	base := createStaffing(f, 1)
	patch := externalCatalogClone(base)
	patch["role"] = "Updated"
	for _, table := range []string{"audit_log", "job_history"} {
		in := confirmStaffing(f, "update", patch)
		f.exec(`CREATE FUNCTION fail_staffing() RETURNS TRIGGER AS $$ BEGIN RAISE EXCEPTION 'synthetic staffing audit rollback'; END $$ LANGUAGE plpgsql;CREATE TRIGGER staff_fail BEFORE INSERT ON ` + table + ` FOR EACH ROW EXECUTE FUNCTION fail_staffing()`)
		receipts := f.count("rental_mcp_mutation_receipts")
		if s, _ := f.call("update", in, "staff-rollback-"+table); s != 500 {
			t.Fatal("audit failure", s)
		}
		var role string
		f.db.QueryRow(`SELECT role FROM job_employees WHERE job_id=1`).Scan(&role)
		if role != "Ton" || f.count("rental_mcp_mutation_receipts") != receipts {
			t.Fatal("partial mutation", role)
		}
		f.exec(`DROP TRIGGER staff_fail ON ` + table + `;DROP FUNCTION fail_staffing()`)
	}
	for _, q := range []string{`DELETE FROM job_employees WHERE job_id=1`, `DELETE FROM employees WHERE id=1`, `UPDATE job_employees SET employee_id=2 WHERE job_id=1`} {
		if _, e := f.db.Exec(q); e == nil {
			t.Fatal("guard missing", q)
		}
	}
	a := confirmStaffing(f, "archive", base)
	if s, p := f.call("archive", a, "staff-retain-archive"); s != 200 {
		t.Fatal(s, p)
	}
	for _, q := range []string{`UPDATE job_employees SET role='edit archived' WHERE job_id=1`, `UPDATE job_employees SET deleted_at=NULL,role='restore changed' WHERE job_id=1`} {
		if _, e := f.db.Exec(q); e == nil {
			t.Fatal("archived guard", q)
		}
	}
	f.exec(`UPDATE jobs SET deleted_at=NOW(),statusid=6 WHERE jobid=1`)
	if _, e := f.db.Exec(`UPDATE job_employees SET deleted_at=NULL WHERE job_id=1`); e == nil {
		t.Fatal("archived job child guard")
	}
}
func TestRentalStaffingNativeRetentionAuditAndDelegatedBypass(t *testing.T) {
	f := newStaffingFixture(t)
	native := func(method, path, body string) int {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)
		return w.Code
	}
	if s := native("POST", "/jobs/1/employees", `{"employee_id":1,"role":"Ton"}`); s != 201 {
		t.Fatal(s)
	}
	if s := native("DELETE", "/jobs/1/employees/1", ""); s != 204 {
		t.Fatal(s)
	}
	var deleted bool
	f.db.QueryRow(`SELECT deleted_at IS NOT NULL FROM job_employees WHERE job_id=1`).Scan(&deleted)
	if !deleted {
		t.Fatal("native harddeleted")
	}
	if s := native("POST", "/jobs/1/employees", `{"employee_id":1,"role":"Licht"}`); s != 201 {
		t.Fatal(s)
	}
	if f.count("audit_log") != 3 || f.count("job_history") != 3 {
		t.Fatal("native audit missing")
	}
	f.exec(`CREATE FUNCTION fail_staff_native() RETURNS TRIGGER AS $$ BEGIN RAISE EXCEPTION 'synthetic native audit rollback'; END $$ LANGUAGE plpgsql;CREATE TRIGGER staff_fail BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_staff_native()`)
	if s := native("DELETE", "/jobs/1/employees/1", ""); s != 500 {
		t.Fatal(s)
	}
	f.db.QueryRow(`SELECT deleted_at IS NOT NULL FROM job_employees WHERE job_id=1`).Scan(&deleted)
	if deleted {
		t.Fatal("native audit not atomic")
	}
	for _, scope := range []any{"cores:rental:create", nil, 123} {
		cookie, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": scope, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		req := httptest.NewRequest("POST", "/jobs/1/employees", strings.NewReader(`{"employee_id":2}`))
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: cookie})
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatal("delegated native bypass", scope, w.Code)
		}
	}
}
func TestRentalStaffingBoundsAndTwoConnectionCollision(t *testing.T) {
	f := newStaffingFixture(t)
	for _, in := range []map[string]any{{"job_id": 0, "employee_id": 1}, {"job_id": 1, "employee_id": 1, "role": nil}, {"job_id": 1, "employee_id": 1, "role": strings.Repeat("ä", 101)}, {"job_id": 1, "employee_id": 1, "extra": true}} {
		if s, _ := f.call("create", in, ""); s != 400 {
			t.Fatal("validation", s, in)
		}
	}
	base := map[string]any{"job_id": 1, "employee_id": 1, "role": strings.Repeat("ä", 100)}
	in := confirmStaffing(f, "create", base)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`LOCK TABLE job_employees IN ROW EXCLUSIVE MODE`); e != nil {
		t.Fatal(e)
	}
	started := time.Now()
	if s, _ := f.call("create", in, "staff-table-collision"); s != 409 {
		t.Fatal("collision", s)
	}
	if time.Since(started) > time.Second {
		t.Fatal("waited reverse locks")
	}
	tx.Rollback()
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	// Independent HTTP requests with the same durable key create exactly one identity/audit.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s, _ := f.call("create", in, "staff-concurrent-samekey"); codes <- s }()
	}
	wg.Wait()
	close(codes)
	for s := range codes {
		if s != 201 {
			t.Fatal("concurrent replay", s)
		}
	}
	if f.count("job_employees") != 1 || f.count("audit_log") != 1 {
		t.Fatal("duplicate concurrent write")
	}
}

func TestRentalStaffingEmployeeActivationLocksBothOrders(t *testing.T) {
	f := newStaffingFixture(t)
	// Deactivation obtains the employee row first: a legacy insert cannot pass the guard.
	first, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer first.Rollback()
	if _, e = first.Exec(`UPDATE employees SET is_active=false WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	second, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer second.Rollback()
	second.Exec(`SET LOCAL lock_timeout='100ms'`)
	if _, e = second.Exec(`INSERT INTO job_employees(job_id,employee_id) VALUES(1,1)`); e == nil {
		t.Fatal("insert did not lock employee")
	}
	second.Rollback()
	if e = first.Commit(); e != nil {
		t.Fatal(e)
	}
	if _, e = f.db.Exec(`INSERT INTO job_employees(job_id,employee_id) VALUES(1,1)`); e == nil {
		t.Fatal("assigned inactive employee")
	}
	f.exec(`UPDATE employees SET is_active=true WHERE id=1`)
	// Insert obtains employee share lock first: deactivation must wait for its commit.
	first, e = f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer first.Rollback()
	if _, e = first.Exec(`INSERT INTO job_employees(job_id,employee_id) VALUES(1,1)`); e != nil {
		t.Fatal(e)
	}
	second, e = f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer second.Rollback()
	second.Exec(`SET LOCAL lock_timeout='100ms'`)
	if _, e = second.Exec(`UPDATE employees SET is_active=false WHERE id=1`); e == nil {
		t.Fatal("deactivation bypassed employee share lock")
	}
	second.Rollback()
	if e = first.Commit(); e != nil {
		t.Fatal(e)
	}
	f.exec(`UPDATE job_employees SET deleted_at=NOW() WHERE job_id=1;UPDATE employees SET is_active=false WHERE id=1`)
	if _, e = f.db.Exec(`UPDATE job_employees SET deleted_at=NULL WHERE job_id=1`); e == nil {
		t.Fatal("restored inactive employee")
	}
}

func TestRentalStaffingSchemaMatchesMigrationAndBoundsBusyStartup(t *testing.T) {
	f := newStaffingFixture(t)
	migration, e := os.ReadFile("../../migrations/054_rental_staffing_lifecycle.sql")
	if e != nil {
		t.Fatal(e)
	}
	if string(migration) != schema.RentalStaffingLifecycleSQL {
		t.Fatal("startup/service migration differ")
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`LOCK TABLE job_employees IN ACCESS EXCLUSIVE MODE`); e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	e = schema.EnsureRentalStaffingLifecycle(f.db)
	if e == nil || time.Since(start) > 7*time.Second {
		t.Fatal("unbounded/bypassed busy-table startup", e, time.Since(start))
	}
}

type staffingCalendarProbe struct {
	db        *sql.DB
	synced    chan uint
	deletes   atomic.Int64
	calls     atomic.Int64
	committed atomic.Bool
}

func (p *staffingCalendarProbe) SyncJobEvent(job uint) {
	var n int
	p.db.QueryRow(`SELECT count(*) FROM audit_log WHERE entity_type='rental_staffing'`).Scan(&n)
	p.committed.Store(n > 0)
	p.calls.Add(1)
	p.synced <- job
}
func (p *staffingCalendarProbe) DeleteJobEvent(uint)            {}
func (p *staffingCalendarProbe) SyncEmployeeEvent(uint, uint)   {}
func (p *staffingCalendarProbe) DeleteEmployeeEvent(uint, uint) { p.deletes.Add(1) }
func TestRentalStaffingCalendarOnlyAfterCommitNeverPreviewOrReplay(t *testing.T) {
	f := newStaffingFixture(t)
	probe := &staffingCalendarProbe{db: f.db, synced: make(chan uint, 5)}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	h := &RentalStaffingMCP{db: f.db, calendar: probe}
	router.POST("/:operation", h.Change)
	f.router = router
	for i, op := range []string{"create", "update", "archive", "restore"} {
		base := map[string]any{"job_id": 1, "employee_id": 1}
		if op == "update" {
			base["role"] = "Updated"
		}
		in := confirmStaffing(f, op, base)
		if probe.calls.Load() != int64(i) {
			t.Fatal("preview scheduled calendar")
		}
		key := "staff-calendar-" + op
		s, out := f.call(op, in, key)
		if s != 200 && s != 201 {
			t.Fatal(s, out)
		}
		select {
		case <-probe.synced:
		case <-time.After(3 * time.Second):
			t.Fatal("postcommit calendar not scheduled")
		}
		if !probe.committed.Load() {
			t.Fatal("calendar preceded audit commit")
		}
		if s, _ = f.call(op, in, key); s != 200 && s != 201 {
			t.Fatal(s)
		}
		if probe.calls.Load() != int64(i+1) {
			t.Fatal("replay rescheduled calendar")
		}
	}
	if probe.deletes.Load() != 1 {
		t.Fatal("archive legacy cleanup", probe.deletes.Load())
	}
}
