package schema

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRentalJobLifecycleVersionsAndHistory(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("dedicated test DB required")
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
	fail := func(query string) {
		t.Helper()
		if _, err := db.Exec(query); err == nil {
			t.Fatal("guard accepted forbidden write", query)
		}
	}
	const ns = "rental_job_lifecycle_test"
	exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
	exec("CREATE SCHEMA " + ns)
	defer exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
	exec("SET search_path TO " + ns)
	exec(`CREATE TABLE jobs(jobid SERIAL PRIMARY KEY,statusid INTEGER DEFAULT 1,description TEXT,deleted_at TIMESTAMP,updated_at TIMESTAMP DEFAULT NOW(),m365_event_id TEXT,updated_by INTEGER)`)
	exec(`CREATE TABLE job_devices(jobid INTEGER,deviceid TEXT,pack_status TEXT,PRIMARY KEY(jobid,deviceid))`)
	exec(`CREATE TABLE job_positions(position_id SERIAL PRIMARY KEY,job_id INTEGER,quantity NUMERIC,updated_at TIMESTAMP DEFAULT NOW())`)
	exec(`CREATE TABLE job_employees(id SERIAL PRIMARY KEY,job_id INTEGER,employee_id INTEGER,m365_event_id TEXT,updated_at TIMESTAMP DEFAULT NOW())`)
	exec(`CREATE TABLE warehouse_tasks(task_id SERIAL PRIMARY KEY,job_id INTEGER,status TEXT,is_archived BOOLEAN NOT NULL DEFAULT false)`)
	exec(`CREATE TABLE cases(caseid SERIAL PRIMARY KEY,current_job_id INTEGER,lifecycle_status TEXT DEFAULT 'active')`)
	if err := EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join("..", "..", "migrations", "048_rental_job_lifecycle.sql"), filepath.Join("..", "..", "..", "migrations", "postgresql", "034_rental_job_lifecycle.sql")} {
		raw, err := os.ReadFile(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || string(raw) != RentalJobLifecycleSQL {
			t.Fatal("startup/migration SQL drift", file, err)
		}
	}
	exec(`INSERT INTO jobs(description) VALUES('retained job')`)
	version := func() time.Time {
		var ts time.Time
		if err := db.QueryRow(`SELECT updated_at FROM jobs WHERE jobid=1`).Scan(&ts); err != nil {
			t.Fatal(err)
		}
		return ts
	}
	before := version()
	exec(`UPDATE jobs SET description='legacy metadata',updated_at='2000-01-01' WHERE jobid=1`)
	if !version().After(before) {
		t.Fatal("legacy writer did not advance precise version")
	}
	before = version()
	exec(`INSERT INTO job_positions(job_id,quantity) VALUES(1,2)`)
	if !version().After(before) {
		t.Fatal("position insertion not versioned")
	}
	before = version()
	exec(`UPDATE job_positions SET quantity=3`)
	if !version().After(before) {
		t.Fatal("position update not versioned")
	}
	before = version()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE job_positions SET quantity=99`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if !version().Equal(before) {
		t.Fatal("unexpected version rollback")
	}
	before = version()
	exec(`INSERT INTO job_devices VALUES(1,'issued fixture','issued')`)
	fail(`UPDATE jobs SET deleted_at=NOW() WHERE jobid=1`)
	exec(`UPDATE job_devices SET pack_status='returned'`)
	exec(`INSERT INTO warehouse_tasks(job_id,status) VALUES(1,'open')`)
	fail(`UPDATE jobs SET deleted_at=NOW() WHERE jobid=1`)
	exec(`UPDATE warehouse_tasks SET status='done'`)
	exec(`INSERT INTO cases(current_job_id) VALUES(1)`)
	fail(`UPDATE jobs SET deleted_at=NOW() WHERE jobid=1`)
	exec(`UPDATE cases SET current_job_id=NULL`)
	fail(`UPDATE jobs SET jobid=7 WHERE jobid=1`)
	fail(`DELETE FROM jobs WHERE jobid=1`)
	fail(`UPDATE jobs SET description='hidden field edit',deleted_at=NOW() WHERE jobid=1`)
	exec(`INSERT INTO job_employees(job_id,employee_id,m365_event_id) VALUES(1,42,'legacy event')`)
	exec(`UPDATE jobs SET statusid=6,deleted_at=NOW() WHERE jobid=1`)
	fail(`UPDATE jobs SET description='archived edit' WHERE jobid=1`)
	fail(`UPDATE jobs SET deleted_at=NULL,description='hidden restore edit' WHERE jobid=1`)
	fail(`UPDATE job_positions SET quantity=4`)
	fail(`DELETE FROM job_positions`)
	fail(`INSERT INTO job_positions(job_id,quantity) VALUES(1,5)`)
	fail(`UPDATE job_employees SET employee_id=43`)
	exec(`UPDATE job_employees SET m365_event_id=NULL,updated_at=NOW()`)
	exec(`UPDATE jobs SET m365_event_id=NULL WHERE jobid=1`)
	if err := EnsureRentalJobLifecycle(db); err != nil {
		t.Fatal("idempotent startup while job is archived", err)
	}
	exec(`UPDATE jobs SET deleted_at=NULL WHERE jobid=1`)
	exec(`UPDATE jobs SET statusid=1 WHERE jobid=1`)
	exec(`UPDATE job_positions SET quantity=4`)
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM job_positions WHERE job_id=1 AND quantity=4`).Scan(&rows); err != nil || rows != 1 {
		t.Fatal("restore lost history", rows, err)
	}
}
