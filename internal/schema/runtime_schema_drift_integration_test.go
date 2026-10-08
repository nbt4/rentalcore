package schema_test

import (
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"

	"go-barcode-webapp/internal/compliance"
	"go-barcode-webapp/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const runtimeSchemaMigrationPath = "../../migrations/051_repair_runtime_schema_drift.sql"

func TestRuntimeSchemaDriftMigrationDefinesRequiredPostgresObjects(t *testing.T) {
	raw, err := os.ReadFile(runtimeSchemaMigrationPath)
	if err != nil {
		t.Fatalf("read runtime schema migration: %v", err)
	}
	migration := string(raw)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS audit_events",
		"CREATE TABLE IF NOT EXISTS gobd_records",
		"CREATE TABLE IF NOT EXISTS retention_policies",
		"CREATE TABLE IF NOT EXISTS invoice_templates",
		"REFERENCES product_packages(id)",
	} {
		if !strings.Contains(migration, required) {
			t.Errorf("migration does not contain %q", required)
		}
	}
	if strings.Contains(migration, "REFERENCES packages(") || strings.Contains(migration, "REFERENCES product_packages(package_id)") {
		t.Error("migration references a non-canonical package table or primary key")
	}

	suiteRaw, err := os.ReadFile("../../../migrations/postgresql/049_rental_schema_drift.sql")
	if err == nil {
		migrationBody := migration[strings.Index(migration, "CREATE TABLE"):]
		suiteMigration := string(suiteRaw)
		suiteBody := suiteMigration[strings.Index(suiteMigration, "CREATE TABLE"):]
		if strings.TrimSpace(migrationBody) != strings.TrimSpace(suiteBody) {
			t.Error("RentalCore and suite migration bodies differ")
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("read suite migration mirror: %v", err)
	}
}

func TestRuntimeSchemaDriftMigrationAppliesTwice(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("dedicated _test DB required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	const namespace = "runtime_schema_drift_test"
	if _, err := db.Exec(`DROP SCHEMA IF EXISTS ` + namespace + ` CASCADE; CREATE SCHEMA ` + namespace + `; SET search_path TO ` + namespace); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	defer db.Exec(`DROP SCHEMA IF EXISTS ` + namespace + ` CASCADE`)

	fixture := `
		CREATE TABLE users (userid SERIAL PRIMARY KEY);
		CREATE TABLE product_packages (id BIGSERIAL PRIMARY KEY);
		CREATE TABLE pdf_extraction_items (item_id BIGSERIAL PRIMARY KEY);
		CREATE TABLE retention_policies (
			id BIGSERIAL PRIMARY KEY,
			document_type VARCHAR(100) NOT NULL,
			retention_period_days INTEGER NOT NULL,
			legal_basis VARCHAR(255) NOT NULL,
			auto_delete BOOLEAN DEFAULT FALSE,
			policy_description TEXT,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO retention_policies
			(document_type, retention_period_days, legal_basis, auto_delete, policy_description)
		VALUES ('invoice', 3650, 'test basis', TRUE, 'legacy description');
	`
	if _, err := db.Exec(fixture); err != nil {
		t.Fatalf("create migration fixture: %v", err)
	}

	raw, err := os.ReadFile(runtimeSchemaMigrationPath)
	if err != nil {
		t.Fatalf("read runtime schema migration: %v", err)
	}
	for run := 1; run <= 2; run++ {
		if _, err := db.Exec(string(raw)); err != nil {
			t.Fatalf("apply runtime schema migration (run %d): %v", run, err)
		}
	}

	for _, table := range []string{"audit_events", "gobd_records", "retention_policies", "invoice_templates"} {
		var exists bool
		if err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, namespace+"."+table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s was not created", table)
		}
	}

	var targetColumn string
	err = db.QueryRow(`
		SELECT ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_schema = tc.constraint_schema
		 AND kcu.constraint_name = tc.constraint_name
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_schema = tc.constraint_schema
		 AND ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_schema = $1
		  AND tc.table_name = 'pdf_extraction_items'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND kcu.column_name = 'mapped_package_id'
		  AND ccu.table_name = 'product_packages'
	`, namespace).Scan(&targetColumn)
	if err != nil {
		t.Fatalf("find package mapping foreign key: %v", err)
	}
	if targetColumn != "id" {
		t.Fatalf("package mapping foreign key targets %q, want id", targetColumn)
	}

	var retentionYears int
	var autoDeleteAfter bool
	var description string
	if err := db.QueryRow(`
		SELECT retention_years, auto_delete_after, description
		FROM retention_policies
		WHERE document_type = 'invoice'
	`).Scan(&retentionYears, &autoDeleteAfter, &description); err != nil {
		t.Fatalf("read migrated legacy retention policy: %v", err)
	}
	if retentionYears != 10 || !autoDeleteAfter || description != "legacy description" {
		t.Fatalf("migrated retention policy = (%d, %t, %q), want (10, true, legacy description)", retentionYears, autoDeleteAfter, description)
	}

	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open migrated schema with gorm: %v", err)
	}
	gobd, err := compliance.NewGoBDCompliance(orm, t.TempDir())
	if err != nil {
		t.Fatalf("initialize GoBD compliance on migrated schema: %v", err)
	}
	if err := gobd.ArchiveDocument("invoice", "test-1", map[string]string{"status": "test"}, 0); err != nil {
		t.Fatalf("archive document on migrated schema: %v", err)
	}

	template := &models.InvoiceTemplate{
		Name:         "Test template",
		HTMLTemplate: "<p>test</p>",
		IsDefault:    true,
		IsActive:     true,
	}
	if err := orm.Create(template).Error; err != nil {
		t.Fatalf("create invoice template on migrated schema: %v", err)
	}
}
