package handlers

import (
	"testing"

	"go-barcode-webapp/internal/models"
)

func TestBuildMappingRows_ProductsAndPackages(t *testing.T) {
	productMappings := []models.PDFProductMapping{
		{MappingID: 1, PDFProductText: "Mikrofon e935", ProductID: 10, MappingType: "manual", UsageCount: 5},
		{MappingID: 2, PDFProductText: "DI-Box passiv", ProductID: 99, MappingType: "fuzzy", UsageCount: 1},
	}
	packageMappings := []models.PDFPackageMapping{
		{MappingID: 3, PDFPackageText: "PA Paket klein", PackageID: 7, MappingType: "manual", UsageCount: 2},
	}
	productNames := map[int]string{10: "Sennheiser e935"}
	packageNames := map[int]string{7: "PA Set S"}

	rows := buildMappingRows(productMappings, packageMappings, productNames, packageNames)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// First row: known product
	if rows[0].TargetType != "product" || rows[0].TargetName != "Sennheiser e935" || rows[0].TargetID != 10 {
		t.Errorf("row[0] unexpected: %+v", rows[0])
	}

	// Second row: unknown product ID falls back to "Product #99"
	if rows[1].TargetName != "Product #99" {
		t.Errorf("row[1] fallback name: got %q, want %q", rows[1].TargetName, "Product #99")
	}

	// Third row: package
	if rows[2].TargetType != "package" || rows[2].TargetName != "PA Set S" || rows[2].TargetID != 7 {
		t.Errorf("row[2] unexpected: %+v", rows[2])
	}
}

func TestBuildMappingRows_Empty(t *testing.T) {
	rows := buildMappingRows(nil, nil, nil, nil)
	if len(rows) != 0 {
		t.Errorf("expected empty, got %d rows", len(rows))
	}
}
