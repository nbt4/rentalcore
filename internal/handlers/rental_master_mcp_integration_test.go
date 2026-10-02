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

func TestRentalMasterOwnerAtomicLifecycle(t *testing.T) {
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
	const ns = "rental_master_owner_test"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, e := db.Exec(query, args...); e != nil {
			t.Fatal(e)
		}
	}
	failSQL := func(query string) {
		t.Helper()
		if _, e := db.Exec(query); e == nil {
			t.Fatal("unsafe legacy writer accepted", query)
		}
	}
	exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE;CREATE SCHEMA ` + ns + `;SET search_path TO ` + ns)
	defer db.Exec(`DROP SCHEMA IF EXISTS ` + ns + ` CASCADE`)
	exec(`CREATE TABLE users(userid INTEGER PRIMARY KEY,is_admin BOOLEAN,is_active BOOLEAN);
 INSERT INTO users VALUES(1,true,true),(2,false,true);
 CREATE TABLE customers(customerid SERIAL PRIMARY KEY,name VARCHAR(255),companyname VARCHAR(255),firstname VARCHAR(100),lastname VARCHAR(100),street VARCHAR(255),housenumber VARCHAR(20),zip VARCHAR(20),city VARCHAR(100),federalstate VARCHAR(100),country VARCHAR(100),phonenumber VARCHAR(50),email VARCHAR(255),customertype VARCHAR(50),is_customer BOOLEAN DEFAULT true,is_supplier BOOLEAN DEFAULT false,notes TEXT,m365_id TEXT,gal_contact_id TEXT,m365_updated_at TIMESTAMP,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
 CREATE TABLE status(statusid INTEGER PRIMARY KEY,status TEXT);INSERT INTO status VALUES(1,'Planung'),(4,'Abgeschlossen');
 CREATE TABLE jobs(jobid SERIAL PRIMARY KEY,customerid INTEGER REFERENCES customers(customerid),statusid INTEGER,deleted_at TIMESTAMP,updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
 CREATE TABLE audit_log(id BIGSERIAL PRIMARY KEY,user_id INTEGER,action TEXT,entity_type TEXT,entity_id TEXT,old_values JSONB,new_values JSONB,timestamp TIMESTAMPTZ);`)
	for i := 0; i < 2; i++ {
		if err = schema.EnsureRentalMasterLifecycle(db); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("test-rental-master-", 3))
	gin.SetMode(gin.TestMode)
	handler := &RentalMasterMCP{db: db}
	user := &models.User{UserID: 1, IsAdmin: true, IsActive: true}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", user) })
	router.POST("/customers/:operation", handler.Customer)
	router.POST("/venues/:operation", handler.Venue)
	invoke := func(kind, op string, in map[string]any, key, scope string) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(in)
		r := httptest.NewRequest("POST", "/"+kind+"s/"+op, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Cores-Origin", "MCP/AI")
		r.Header.Set("Idempotency-Key", key)
		token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": scope, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		if e != nil {
			t.Fatal(e)
		}
		r.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		out := map[string]any{}
		if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	scope := func(op string) string {
		if op == "revert_update" {
			op = "update"
		}
		if op == "restore" {
			op = "archive"
		}
		return "cores:rental:" + op
	}
	prepare := func(kind, op string, in map[string]any) (map[string]any, map[string]any) {
		t.Helper()
		copy := map[string]any{}
		for k, v := range in {
			copy[k] = v
		}
		copy["preview"] = true
		status, p := invoke(kind, op, copy, "", scope(op))
		if status != 200 {
			t.Fatalf("preview %d %#v", status, p)
		}
		delete(copy, "preview")
		copy["expected_updated_at"] = p["expected_updated_at"]
		copy["expected_context"] = p["expected_context"]
		copy["confirmation_text"] = p["required_confirmation_text"]
		copy["confirm_change"] = true
		if op == "revert_update" {
			copy["audit_id"] = p["expected_audit_id"]
		}
		return copy, p
	}
	execute := func(kind, op string, in map[string]any, key string) map[string]any {
		t.Helper()
		status, out := invoke(kind, op, in, key, scope(op))
		if status < 200 || status > 299 || out["record"] == nil {
			t.Fatalf("execute %d %#v", status, out)
		}
		return out
	}
	has := func(p map[string]any, key string) bool {
		for _, v := range p["required_fields"].([]any) {
			if v == key {
				return true
			}
		}
		return false
	}
	_, p := prepare("customer", "create", map[string]any{})
	if !has(p, "name") {
		t.Fatal(p)
	}
	create, p := prepare("customer", "create", map[string]any{"company_name": "Lifecycle company", "first_name": "Contact", "last_name": "Person", "street": "Business Street", "house_number": "2a", "zip": "00123", "city": "Berlin", "federal_state": "Berlin", "country": "Deutschland", "phone": "030 1234", "email": "contact@example.test", "customer_type": "Unternehmen", "is_supplier": true, "notes": "private business note"})
	if p["ready_to_execute"] != true || p["draft"].(map[string]any)["is_customer"] != true {
		t.Fatal(p)
	}
	status, _ := invoke("customer", "create", create, "rental-master-wrong-scope", "cores:rental:update")
	if status != 403 {
		t.Fatal(status)
	}
	user.IsAdmin = false
	status, _ = invoke("customer", "create", create, "rental-master-nonadmin", scope("create"))
	if status != 403 {
		t.Fatal(status)
	}
	user.IsAdmin = true
	exec(`UPDATE customers SET name='nothing' WHERE false`)
	created := execute("customer", "create", create, "rental-master-create")
	record := created["record"].(map[string]any)
	id := int(record["id"].(float64))
	if record["zip"] != "00123" || record["notes"] != "private business note" || record["is_supplier"] != true {
		t.Fatal(record)
	}
	if replay := execute("customer", "create", create, "rental-master-create"); !reflect.DeepEqual(created, replay) {
		t.Fatal("create replay differs")
	}
	exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	status, _ = invoke("customer", "create", create, "rental-master-create", scope("create"))
	if status != 403 {
		t.Fatal("revoked rights replay", status)
	}
	exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	_, p = prepare("customer", "create", map[string]any{"company_name": "Lifecycle company"})
	if !has(p, "review_duplicate_and_allow_explicitly") {
		t.Fatal(p)
	}
	_, p = prepare("customer", "create", map[string]any{"company_name": "Lifecycle company", "allow_duplicate": true})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	update, p := prepare("customer", "update", map[string]any{"id": id, "phone": "", "city": "Hamburg"})
	if p["draft"].(map[string]any)["notes"] != "private business note" || p["draft"].(map[string]any)["phone"] != nil {
		t.Fatal(p)
	}
	exec(`UPDATE customers SET city='Raw concurrent metadata' WHERE customerid=$1`, id)
	status, out := invoke("customer", "update", update, "rental-master-stale", scope("update"))
	if status != 200 || !has(out, "expected_updated_at") || !has(out, "expected_context") {
		t.Fatal(status, out)
	}
	update, p = prepare("customer", "update", map[string]any{"id": id, "phone": "", "city": "Hamburg"})
	exec(`CREATE FUNCTION reject_master_audit() RETURNS TRIGGER AS $$ BEGIN IF NEW.action='rental.customer.update' THEN RAISE EXCEPTION 'final audit failure';END IF;RETURN NEW;END;$$ LANGUAGE plpgsql;CREATE TRIGGER reject_master_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION reject_master_audit()`)
	status, _ = invoke("customer", "update", update, "rental-master-retry", scope("update"))
	if status != 500 {
		t.Fatal(status)
	}
	_, again := prepare("customer", "update", map[string]any{"id": id, "phone": "", "city": "Hamburg"})
	if again["expected_context"] != p["expected_context"] {
		t.Fatal("failed audit left partial change", again)
	}
	exec(`DROP TRIGGER reject_master_audit ON audit_log`)
	updated := execute("customer", "update", update, "rental-master-retry")
	if updated["record"].(map[string]any)["phone"] != nil {
		t.Fatal(updated)
	}
	venue, p := prepare("venue", "create", map[string]any{"name": "Venue one", "street": "Event Street", "house_number": "12", "zip": "00001", "city": "Berlin", "contact_name": "Business contact", "phone": "030 222", "email": "venue@example.test", "notes": "private venue note"})
	if p["ready_to_execute"] != true {
		t.Fatal(p)
	}
	v := execute("venue", "create", venue, "rental-master-venue-create")
	vid := int(v["record"].(map[string]any)["id"].(float64))
	exec(`INSERT INTO jobs(customerid,statusid,venue_id) VALUES($1,1,$2)`, id, vid)
	for _, item := range []struct {
		kind string
		id   int
	}{{"customer", id}, {"venue", vid}} {
		_, p = prepare(item.kind, "archive", map[string]any{"id": item.id})
		if !has(p, "active_jobs") {
			t.Fatal(p)
		}
		spec := masterSpec(item.kind)
		failSQL(fmt.Sprintf("UPDATE %s SET is_archived=true WHERE %s=%d", spec.table, spec.pk, item.id))
	}
	exec(`UPDATE jobs SET statusid=4`)
	for _, item := range []struct {
		kind string
		id   int
	}{{"customer", id}, {"venue", vid}} {
		a, p := prepare(item.kind, "archive", map[string]any{"id": item.id})
		if p["ready_to_execute"] != true {
			t.Fatal(p)
		}
		archived := execute(item.kind, "archive", a, "rental-master-"+item.kind+"-archive")
		if archived["record"].(map[string]any)["is_archived"] != true {
			t.Fatal(archived)
		}
		spec := masterSpec(item.kind)
		failSQL(fmt.Sprintf("UPDATE %s SET notes='Forbidden' WHERE %s=%d", spec.table, spec.pk, item.id))
		failSQL(fmt.Sprintf("UPDATE %s SET is_archived=false,notes='Hidden restore edit' WHERE %s=%d", spec.table, spec.pk, item.id))
		failSQL(fmt.Sprintf("DELETE FROM %s WHERE %s=%d", spec.table, spec.pk, item.id))
		_, p = prepare(item.kind, "create", map[string]any{"name": masterIdentity(item.kind, archived["record"].(map[string]any))})
		if !has(p, "restoration_required") {
			t.Fatal(p)
		}
		r, p := prepare(item.kind, "restore", map[string]any{"id": item.id})
		if p["ready_to_execute"] != true {
			t.Fatal(p)
		}
		restored := execute(item.kind, "restore", r, "rental-master-"+item.kind+"-restore")
		if restored["record"].(map[string]any)["notes"] != archived["record"].(map[string]any)["notes"] {
			t.Fatal(restored)
		}
		if replay := execute(item.kind, "archive", a, "rental-master-"+item.kind+"-archive"); !reflect.DeepEqual(replay, archived) {
			t.Fatal("historical durable replay differs")
		}
	}
	failSQL(fmt.Sprintf("UPDATE customers SET customerid=9999 WHERE customerid=%d", id))
	failSQL(fmt.Sprintf("UPDATE venues SET id=9999 WHERE id=%d", vid))
	failSQL(fmt.Sprintf("UPDATE customers SET is_archived=true,notes='Hidden archive edit' WHERE customerid=%d", id))
	var audits, receipts int
	if err = db.QueryRow(`SELECT count(*) FROM audit_log`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM rental_mcp_mutation_receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if audits != 7 || receipts != 7 {
		t.Fatal(audits, receipts)
	}
	// Missing status is conservatively active, and legacy job writers cannot
	// reactivate historic references to an archived master.
	exec(`UPDATE jobs SET statusid=NULL`)
	_, p = prepare("venue", "archive", map[string]any{"id": vid})
	if !has(p, "active_jobs") {
		t.Fatal(p)
	}
	exec(`UPDATE jobs SET deleted_at=CURRENT_TIMESTAMP`)
	a, _ := prepare("venue", "archive", map[string]any{"id": vid})
	execute("venue", "archive", a, "rental-master-final-archive")
	failSQL(`UPDATE jobs SET deleted_at=NULL`)
	// Existing legacy inconsistencies can be inspected and repaired by restore;
	// a no-op status normalization must not prevent the service from starting.
	exec(`UPDATE jobs SET statusid=1;ALTER TABLE jobs DISABLE TRIGGER jobs_guard_master_lifecycle;UPDATE jobs SET deleted_at=NULL;ALTER TABLE jobs ENABLE TRIGGER jobs_guard_master_lifecycle`)
	exec(`UPDATE jobs SET statusid=statusid`)
	failSQL(`UPDATE jobs SET statusid=NULL`)
	_, p = prepare("customer", "revert_update", map[string]any{"id": id})
	if !has(p, "own_last_update") {
		t.Fatal("lifecycle action became a field undo", p)
	}
	update, _ = prepare("customer", "update", map[string]any{"id": id, "phone": "030 undo", "notes": "temporary private note", "country": "UndoLand"})
	execute("customer", "update", update, "rental-master-before-undo")
	undo, p := prepare("customer", "revert_update", map[string]any{"id": id})
	if p["ready_to_execute"] != true || p["draft"].(map[string]any)["country"] != "Deutschland" || p["draft"].(map[string]any)["phone"] != nil {
		t.Fatal(p)
	}
	auditID := int(p["expected_audit_id"].(float64))
	var originalBefore string
	if err = db.QueryRow(`SELECT old_values::text FROM audit_log WHERE id=$1`, auditID).Scan(&originalBefore); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE audit_log SET old_values=jsonb_set(old_values,'{email}','"Invalid legacy address"'::jsonb) WHERE id=$1`, auditID)
	_, invalidHistory := prepare("customer", "revert_update", map[string]any{"id": id})
	if !has(invalidHistory, "valid_update_history") {
		t.Fatal("invalid legacy before values became undoable", invalidHistory)
	}
	exec(`UPDATE audit_log SET old_values=$2::jsonb WHERE id=$1`, auditID, originalBefore)
	exec(`UPDATE audit_log SET user_id=2 WHERE id=$1`, auditID)
	_, denied := prepare("customer", "revert_update", map[string]any{"id": id})
	if !has(denied, "own_last_update") {
		t.Fatal("another actor update became undoable", denied)
	}
	exec(`UPDATE audit_log SET user_id=1 WHERE id=$1`, auditID)
	badAudit := map[string]any{}
	for k, v := range undo {
		badAudit[k] = v
	}
	badAudit["audit_id"] = auditID + 1
	status, out = invoke("customer", "revert_update", badAudit, "rental-master-wrong-audit", scope("revert_update"))
	if status != 200 || !has(out, "audit_id") {
		t.Fatal(status, out)
	}
	status, _ = invoke("customer", "revert_update", map[string]any{"id": id, "notes": "Injected undo edit"}, "", scope("revert_update"))
	if status != 400 {
		t.Fatal("arbitrary field revert accepted", status)
	}
	exec(`CREATE FUNCTION reject_revert_audit() RETURNS TRIGGER AS $$ BEGIN IF NEW.action='rental.customer.revert_update' THEN RAISE EXCEPTION 'final undo audit failure';END IF;RETURN NEW;END;$$ LANGUAGE plpgsql;CREATE TRIGGER reject_revert_audit BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION reject_revert_audit()`)
	status, _ = invoke("customer", "revert_update", undo, "rental-master-undo-retry", scope("revert_update"))
	if status != 500 {
		t.Fatal(status)
	}
	_, again = prepare("customer", "revert_update", map[string]any{"id": id})
	if again["expected_context"] != p["expected_context"] {
		t.Fatal("undo audit failure changed fields", again)
	}
	exec(`DROP TRIGGER reject_revert_audit ON audit_log`)
	reverted := execute("customer", "revert_update", undo, "rental-master-undo-retry")
	afterRevert := reverted["record"].(map[string]any)
	if afterRevert["country"] != "Deutschland" || afterRevert["notes"] != "private business note" || afterRevert["phone"] != nil || int(reverted["reverted_audit_id"].(float64)) != auditID {
		t.Fatal(reverted)
	}
	if replay := execute("customer", "revert_update", undo, "rental-master-undo-retry"); !reflect.DeepEqual(replay, reverted) {
		t.Fatal("undo replay differs")
	}
	_, p = prepare("customer", "revert_update", map[string]any{"id": id})
	if !has(p, "own_last_update") {
		t.Fatal("recursive undo accepted", p)
	}
	update, _ = prepare("customer", "update", map[string]any{"id": id, "city": "New field edit"})
	execute("customer", "update", update, "rental-master-raw-undo-check")
	exec(`UPDATE customers SET city='Changed by another writer' WHERE customerid=$1`, id)
	_, p = prepare("customer", "revert_update", map[string]any{"id": id})
	if !has(p, "unchanged_update_version") {
		t.Fatal("legacy overwrite could be silently undone", p)
	}
	// Explicitly distinct same-named records can both be archived and then
	// restored after reviewing the other archived identity. They are not deleted
	// or silently recreated to escape duplicate handling.
	twinIDs := []int{}
	for i := 0; i < 2; i++ {
		args, _ := prepare("venue", "create", map[string]any{"name": "Intentional twin venue", "allow_duplicate": i == 1})
		r := execute("venue", "create", args, fmt.Sprintf("rental-master-twin-create-%d", i))
		twinIDs = append(twinIDs, int(r["record"].(map[string]any)["id"].(float64)))
	}
	for _, twin := range twinIDs {
		args, _ := prepare("venue", "archive", map[string]any{"id": twin})
		execute("venue", "archive", args, fmt.Sprintf("rental-master-twin-archive-%d", twin))
	}
	_, p = prepare("venue", "create", map[string]any{"name": "Intentional twin venue", "allow_duplicate": true})
	if !has(p, "restoration_required") {
		t.Fatal("archived identity recreated", p)
	}
	for _, twin := range twinIDs {
		_, p = prepare("venue", "restore", map[string]any{"id": twin})
		if !has(p, "review_duplicate_and_allow_explicitly") {
			t.Fatal(p)
		}
		args, ready := prepare("venue", "restore", map[string]any{"id": twin, "allow_duplicate": true})
		if ready["ready_to_execute"] != true {
			t.Fatal(ready)
		}
		execute("venue", "restore", args, fmt.Sprintf("rental-master-twin-restore-%d", twin))
	}
}
