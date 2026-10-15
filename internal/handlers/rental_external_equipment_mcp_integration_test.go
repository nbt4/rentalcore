package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRentalExternalCatalogAtomicCreateAndGuards(t *testing.T) {
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
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DROP SCHEMA IF EXISTS rental_catalog_create_test CASCADE;CREATE SCHEMA rental_catalog_create_test;SET search_path TO rental_catalog_create_test;
 CREATE TABLE users(userid INT PRIMARY KEY,is_admin BOOL,is_active BOOL);INSERT INTO users VALUES(1,true,true);
 CREATE TABLE customers(customerid INT PRIMARY KEY,name TEXT,companyname TEXT,firstname TEXT,lastname TEXT,is_supplier BOOL,is_archived BOOL);INSERT INTO customers VALUES(1,'Synthetic Supplier',NULL,NULL,NULL,true,false),(2,'Archived',NULL,NULL,NULL,true,true),(3,'Customer',NULL,NULL,NULL,false,false);
 CREATE TABLE rental_equipment(id SERIAL PRIMARY KEY,name VARCHAR(255) NOT NULL,supplier VARCHAR(255),supplier_id INT REFERENCES customers(customerid),category VARCHAR(100),description TEXT,notes TEXT,rental_price NUMERIC(10,2),customer_price NUMERIC(10,2),is_active BOOL DEFAULT true,created_at TIMESTAMP DEFAULT NOW(),updated_at TIMESTAMP DEFAULT NOW());
 CREATE TABLE rental_mcp_mutation_receipts(id BIGSERIAL PRIMARY KEY,user_id INT,operation TEXT,key_hash TEXT,request_hash TEXT,response JSONB,status_code INT,UNIQUE(user_id,operation,key_hash));
 CREATE TABLE audit_log(id BIGSERIAL PRIMARY KEY,user_id INT,action TEXT,entity_type TEXT,entity_id TEXT,old_values JSONB,new_values JSONB,timestamp TIMESTAMPTZ);`)
	defer db.Exec(`DROP SCHEMA IF EXISTS rental_catalog_create_test CASCADE`)
	t.Setenv("CORES_JWT_SECRET", strings.Repeat("catalog-test-", 4))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", &models.User{UserID: 1}) })
	handler := &RentalExternalEquipmentMCP{db: db}
	router.POST("/create", handler.Create)
	invoke := func(in map[string]any, key, scope string, financial bool) (int, map[string]any) {
		raw, _ := json.Marshal(in)
		req := httptest.NewRequest("POST", "/create", bytes.NewReader(raw))
		req.Header.Set("X-Cores-Origin", "MCP/AI")
		req.Header.Set("Idempotency-Key", key)
		token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 1, "mcp_scope": scope, "mcp_financial": financial, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(os.Getenv("CORES_JWT_SECRET")))
		if e != nil {
			t.Fatal(e)
		}
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		out := map[string]any{}
		if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(w.Body.String())
		}
		return w.Code, out
	}
	call := func(in map[string]any, key string) (int, map[string]any) {
		return invoke(in, key, "cores:rental:create", true)
	}
	draft := func(name string) map[string]any {
		return map[string]any{"name": name, "category": "Audio", "supplier": "Synthetic Supplier", "rental_price": 12.35, "customer_price": 21.50, "description": "Synthetic catalog test", "notes": "Local fixture"}
	}
	confirmed := func(in map[string]any) map[string]any {
		t.Helper()
		status, p := call(in, "")
		if status != 200 || p["ready_to_execute"] != true {
			t.Fatalf("preview %d: %v", status, p)
		}
		out := map[string]any{}
		for k, v := range in {
			out[k] = v
		}
		out["confirm_change"] = true
		out["expected_context"] = p["expected_context"]
		out["confirmation_text"] = p["required_confirmation_text"]
		return out
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if e := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	// Missing values never turn into free rentals; explicit zero remains valid.
	for _, field := range []string{"name", "category", "supplier", "rental_price", "customer_price"} {
		in := draft("Missing " + field)
		delete(in, field)
		status, p := call(in, "")
		if status != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p["required_fields"]), field) {
			t.Fatalf("missing %s: %d %v", field, status, p)
		}
	}
	for _, price := range []float64{-1, 1.001, 100000000} {
		in := draft("Invalid price")
		in["rental_price"] = price
		if s, _ := call(in, ""); s != 400 {
			t.Fatal("bad price accepted", price, s)
		}
	}
	long := draft("Long business fields")
	long["description"] = strings.Repeat("€", 10000)
	long["notes"] = strings.Repeat("😀", 10000)
	if s, p := call(long, ""); s != 200 || p["ready_to_execute"] != true {
		t.Fatal("valid long fields rejected", s, p)
	}
	in := draft("Synthetic rental")
	in["supplier_id"] = 1
	create := confirmed(in)
	if count("rental_equipment") != 0 || count("audit_log") != 0 || count("rental_mcp_mutation_receipts") != 0 {
		t.Fatal("preview wrote data")
	}
	badPhrase := map[string]any{}
	for k, v := range create {
		badPhrase[k] = v
	}
	badPhrase["confirmation_text"] = "CREATE"
	if s, _ := call(badPhrase, "catalog-bad-phrase"); s != 428 {
		t.Fatal("unbound confirmation accepted", s)
	}
	if s, _ := call(create, ""); s != 428 {
		t.Fatal("missing idempotency accepted", s)
	}
	status, out := call(create, "catalog-create-one")
	if status != 201 {
		t.Fatal(status, out)
	}
	record := out["equipment"].(map[string]any)
	if record["name"] != in["name"] || record["rental_price"] != 12.35 || record["customer_price"] != 21.50 || record["supplier_id"] != float64(1) || record["is_active"] != true {
		t.Fatal(record)
	}
	if count("rental_equipment") != 1 || count("audit_log") != 1 || count("rental_mcp_mutation_receipts") != 1 {
		t.Fatal("atomic records missing")
	}
	// Durable retry ignores changed catalog context but rechecks present owner rights.
	if s, replay := call(create, "catalog-create-one"); s != 201 || stringMustJSON(replay) != stringMustJSON(out) {
		t.Fatal("replay mismatch", s, replay)
	}
	create["notes"] = "Changed request"
	if s, _ := call(create, "catalog-create-one"); s != 409 {
		t.Fatal("key reuse accepted", s)
	}
	create["notes"] = in["notes"]
	exec(`UPDATE users SET is_admin=false WHERE userid=1`)
	if s, _ := call(create, "catalog-create-one"); s != 403 {
		t.Fatal("replay after revoked admin", s)
	}
	exec(`UPDATE users SET is_admin=true WHERE userid=1`)
	for _, rights := range []struct {
		scope     string
		financial bool
	}{{"cores:rental:update", true}, {"cores:rental:create", false}} {
		if s, _ := invoke(create, "catalog-create-one", rights.scope, rights.financial); s != 403 {
			t.Fatal("wrong delegation", s)
		}
	}
	// Exact identity is blocked even when allow_similar is set or the item is inactive.
	for _, active := range []bool{true, false} {
		exec(`UPDATE rental_equipment SET is_active=$1`, active)
		duplicate := draft("synthetic RENTAL")
		duplicate["allow_similar"] = true
		s, p := call(duplicate, "")
		if s != 200 || p["ready_to_execute"] != false || !strings.Contains(stringMustJSON(p), "reuse_existing_equipment") {
			t.Fatal("duplicate accepted", s, p)
		}
	}
	similar := draft("Synthetic")
	if _, p := call(similar, ""); p["ready_to_execute"] != false {
		t.Fatal("unreviewed similar item accepted")
	}
	similar["allow_similar"] = true
	zero := draft("Zero rental")
	zero["rental_price"] = 0
	zero["customer_price"] = 0
	if s, p := call(confirmed(zero), "catalog-free-rental"); s != 201 {
		t.Fatal("explicit zero refused", s, p)
	}
	// Optional supplier IDs are canonical, active supplier contacts; mismatches fail.
	for _, id := range []int{2, 3, 999} {
		bad := draft("Bad supplier")
		bad["supplier_id"] = id
		if _, p := call(bad, ""); p["ready_to_execute"] != false {
			t.Fatal("invalid supplier accepted", id, p)
		}
	}
	mismatch := draft("Mismatched supplier")
	mismatch["supplier_id"] = 1
	mismatch["supplier"] = "Other supplier"
	if _, p := call(mismatch, ""); p["ready_to_execute"] != false {
		t.Fatal("supplier name mismatch")
	}
	supplierOnly := draft("Supplier by ID")
	supplierOnly["supplier_id"] = 1
	delete(supplierOnly, "supplier")
	stale := confirmed(supplierOnly)
	exec(`UPDATE customers SET name='Renamed Supplier' WHERE customerid=1`)
	if _, p := call(stale, "catalog-stale-supplier"); p["ready_to_execute"] != false {
		t.Fatal("supplier change ignored", p)
	}
	exec(`UPDATE customers SET name='Synthetic Supplier' WHERE customerid=1`)
	// An intervening duplicate invalidates the preview without creating a receipt.
	stale = confirmed(draft("Concurrent rental"))
	exec(`INSERT INTO rental_equipment(name,supplier,category) VALUES('Concurrent rental','Synthetic Supplier','Audio')`)
	if _, p := call(stale, "catalog-stale-duplicate"); p["ready_to_execute"] != false {
		t.Fatal("stale duplicates ignored", p)
	}
	// Audit failure rolls back both the item and durable receipt, and allows retry.
	retry := confirmed(draft("Rollback rental"))
	before := count("rental_equipment")
	receipts := count("rental_mcp_mutation_receipts")
	exec(`ALTER TABLE audit_log ADD CONSTRAINT fail_catalog_audit CHECK(action<>'rental.external_equipment.create') NOT VALID`)
	if status, _ := call(retry, "catalog-rollback-retry"); status != 500 {
		t.Fatal("audit failure ignored", status)
	}
	if count("rental_equipment") != before || count("rental_mcp_mutation_receipts") != receipts {
		t.Fatal("audit rollback incomplete")
	}
	exec(`ALTER TABLE audit_log DROP CONSTRAINT fail_catalog_audit`)
	if status, p := call(retry, "catalog-rollback-retry"); status != 201 {
		t.Fatal("retry failed", status, p)
	}
	// Two concurrent confirmed drafts create exactly one item.
	// Use two actual connections with the same isolated schema, so the table
	// lock (rather than a single-connection pool) serializes the requests.
	q := u.Query()
	q.Set("search_path", "rental_catalog_create_test")
	u.RawQuery = q.Encode()
	raceDB, e := sql.Open("pgx", u.String())
	if e != nil {
		t.Fatal(e)
	}
	defer raceDB.Close()
	raceDB.SetMaxOpenConns(2)
	handler.db = raceDB
	first := confirmed(draft("Race rental"))
	second := confirmed(draft("Race rental"))
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i, input := range []map[string]any{first, second} {
		wg.Add(1)
		go func(i int, input map[string]any) {
			defer wg.Done()
			key := []string{"catalog-race-first", "catalog-race-second"}[i]
			s, _ := call(input, key)
			statuses <- s
		}(i, input)
	}
	wg.Wait()
	close(statuses)
	created := 0
	for s := range statuses {
		if s == 201 {
			created++
		} else if s != 200 {
			t.Fatal("unexpected race result", s)
		}
	}
	if created != 1 {
		t.Fatal("race created", created)
	}
}

func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
