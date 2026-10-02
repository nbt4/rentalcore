package repository

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProductDependencySuggestionsRetainArchives(t *testing.T) {
	dsn := os.Getenv("RENTALCORE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable _test PostgreSQL required")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("dedicated _test database required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	const schema = "rental_dependency_lifecycle_test"
	if err = db.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE;CREATE SCHEMA ` + schema + `;SET search_path TO ` + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
	fixture := `CREATE TABLE products(productid INT PRIMARY KEY,name TEXT,is_accessory BOOL,is_consumable BOOL,generic_barcode TEXT,count_type_id INT,stock_quantity NUMERIC,lifecycle_status TEXT);
 CREATE TABLE count_types(count_type_id INT PRIMARY KEY,abbreviation TEXT);
 CREATE TABLE product_dependencies(id INT PRIMARY KEY,product_id INT,dependency_product_id INT,is_optional BOOL,default_quantity NUMERIC,notes TEXT,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,lifecycle_status TEXT);
 INSERT INTO products VALUES(1,'Root',false,false,'ROOT',1,1,'active'),(2,'Active accessory',true,false,'ACTIVE',1,2,'active'),(3,'Archived accessory',true,false,'ARCHIVED',1,3,'archived');
 INSERT INTO count_types VALUES(1,'Stk');
 INSERT INTO product_dependencies(id,product_id,dependency_product_id,is_optional,default_quantity,notes,lifecycle_status) VALUES(1,1,2,true,2.5,'Retained relationship','active'),(2,1,3,true,1,NULL,'active'),(3,1,2,false,4,'Retained archive','archived');`
	if err = db.Exec(fixture).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAccessoriesConsumablesRepository(&Database{DB: db})
	rows, err := repo.GetProductDependencies(1)
	if err != nil || len(rows) != 1 || rows[0].ID != 1 || rows[0].DefaultQuantity != 2.5 {
		t.Fatalf("active suggestions: %#v %v", rows, err)
	}
	if err = db.Exec(`UPDATE products SET lifecycle_status='archived' WHERE productid=1`).Error; err != nil {
		t.Fatal(err)
	}
	rows, err = repo.GetProductDependencies(1)
	if err != nil || len(rows) != 0 {
		t.Fatalf("inactive source suggestions: %#v %v", rows, err)
	}
	if err = db.Exec(`UPDATE products SET lifecycle_status='active';UPDATE product_dependencies SET lifecycle_status='active' WHERE id=3`).Error; err != nil {
		t.Fatal(err)
	}
	rows, err = repo.GetProductDependencies(1)
	if err != nil || len(rows) != 3 {
		t.Fatalf("restored suggestions: %#v %v", rows, err)
	}
	var count int64
	if err = db.Table("product_dependencies").Count(&count).Error; err != nil || count != 3 {
		t.Fatal("history removed", count, err)
	}
}
