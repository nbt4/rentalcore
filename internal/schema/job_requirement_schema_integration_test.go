package schema_test

import (
	"os"
	"testing"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRequirementSourceMigrationAndReconcile(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set RENTALCORE_TEST_POSTGRES_DSN to run the database integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec("CREATE SCHEMA job_requirement_integration").Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA job_requirement_integration CASCADE")
	if err := db.Exec("SET search_path TO job_requirement_integration").Error; err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE jobs (jobid SERIAL PRIMARY KEY)`,
		`CREATE TABLE job_product_requirements (
			id BIGSERIAL PRIMARY KEY, job_id INTEGER NOT NULL, product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL CHECK (quantity > 0), created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE (job_id, product_id))`,
		`CREATE TABLE job_positions (
			position_id BIGSERIAL PRIMARY KEY, job_id INTEGER NOT NULL, product_id INTEGER,
			position_type TEXT NOT NULL, quantity NUMERIC NOT NULL)`,
		`INSERT INTO job_product_requirements (job_id, product_id, quantity) VALUES (1, 10, 5), (1, 20, 2)`,
		`INSERT INTO job_positions (job_id, product_id, position_type, quantity)
		 VALUES (1, 10, 'product', 2), (1, 10, 'product', 1), (1, 30, 'product', 4)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := schema.EnsureJobRequirementSources(sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := schema.EnsureJobWorkflowColumns(sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := schema.EnsureJobWorkflowColumns(sqlDB); err != nil {
		t.Fatalf("workflow migration must be idempotent: %v", err)
	}
	if err := db.Exec("INSERT INTO jobs (jobid) VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	var revision int
	if err := db.Raw("SELECT revision FROM jobs WHERE jobid = 1").Scan(&revision).Error; err != nil || revision != 1 {
		t.Fatalf("new jobs must start at revision 1: revision %d, error %v", revision, err)
	}
	if err := schema.EnsureJobRequirementSources(sqlDB); err != nil {
		t.Fatalf("migration must be idempotent: %v", err)
	}
	repo := repository.NewRequirementRepository(&repository.Database{DB: db})
	assertReq := func(productID uint, quantity, manual, position int) {
		t.Helper()
		var req models.JobProductRequirement
		if err := db.Where("job_id = 1 AND product_id = ?", productID).First(&req).Error; err != nil {
			t.Fatal(err)
		}
		if req.Quantity != quantity || req.ManualQuantity != manual || req.PositionQuantity != position {
			t.Fatalf("product %d: got total/manual/position %d/%d/%d, want %d/%d/%d", productID,
				req.Quantity, req.ManualQuantity, req.PositionQuantity, quantity, manual, position)
		}
	}
	assertReq(10, 5, 2, 3)
	assertReq(20, 2, 2, 0)
	assertReq(30, 4, 0, 4)

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM job_positions WHERE product_id = 10").Error; err != nil {
			return err
		}
		return repo.ReconcilePositionRequirements(tx, 1)
	}); err != nil {
		t.Fatal(err)
	}
	assertReq(10, 2, 2, 0)
	assertReq(20, 2, 2, 0)
	assertReq(30, 4, 0, 4)

	if err := repo.SaveRequirements(1, []models.JobProductRequirement{{ProductID: 20, Quantity: 3}}); err != nil {
		t.Fatal(err)
	}
	assertReq(20, 3, 3, 0)
	assertReq(30, 4, 0, 4)
	var count int64
	if err := db.Model(&models.JobProductRequirement{}).Where("job_id = 1 AND product_id = 10").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("removed manual requirement remains: count %d, error %v", count, err)
	}
}
