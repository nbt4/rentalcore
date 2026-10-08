package handlers

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEnsurePackageMappingFKReferencesProductPackagesPrimaryKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_schema = tc.constraint_schema
		 AND kcu.constraint_name = tc.constraint_name
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_schema = tc.constraint_schema
		 AND ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public'
		  AND tc.table_name = 'pdf_extraction_items'
		  AND kcu.column_name = 'mapped_package_id'
		  AND ccu.table_schema = 'public'
		  AND ccu.table_name = 'product_packages'
		  AND ccu.column_name = 'id'
	`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(`(?s)ALTER TABLE pdf_extraction_items.*REFERENCES product_packages\(id\).*ON DELETE SET NULL`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensurePackageMappingFK(db); err != nil {
		t.Fatalf("ensurePackageMappingFK() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
