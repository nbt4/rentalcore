# PDF Mapping Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full PDF mapping workflow — MappingModal in JobsPage, new backend endpoints, and a redesigned mapping management page with 3 tabs and confidence bars.

**Architecture:** The existing `PdfImportBanner` in `JobsPage.tsx` already handles upload → OCR poll → auto-map. We slot a new `MappingModal.tsx` component between auto-map completion and the `onApply` call. The backend gets 6 new endpoints added directly to `pdf_handler.go`. The `mapping_management.html` template is rewritten in-place with tabs, confidence bars, and a "+ Neu" form.

**Tech Stack:** Go/Gin (backend), React/TypeScript/Tailwind (frontend), GORM (DB), PostgreSQL

---

## File Map

| File | Action | Purpose |
|------|---------|---------|
| `internal/handlers/pdf_handler.go` | Modify | Add 6 new handlers, extend `MappingRow` + `buildAllMappingRows`, extend `GetAllMappingsAPI` |
| `cmd/server/main.go` | Modify | Register 6 new routes |
| `web/src/components/MappingModal.tsx` | Create | Full mapping modal component |
| `web/src/pages/JobsPage.tsx` | Modify | Replace silent auto-map with MappingModal |
| `web/templates/mapping_management.html` | Rewrite | 3 tabs, confidence bars, "+ Neu" form, customer CRUD |

---

## Task 1: Extend MappingRow and add customer support to GetAllMappingsAPI

**Files:**
- Modify: `internal/handlers/pdf_handler.go:3180-3295`

The `MappingRow` struct is missing `ConfidenceScore`. `buildMappingRows` doesn't handle customer mappings. `GetAllMappingsAPI` doesn't support `?type=` filter or customer mappings.

- [ ] **Step 1: Add ConfidenceScore to MappingRow**

Find this block (line ~3181):
```go
type MappingRow struct {
	MappingID   uint64 `json:"mapping_id"`
	OCRText     string `json:"ocr_text"`
	TargetName  string `json:"target_name"`
	TargetType  string `json:"target_type"` // "product" or "package"
	TargetID    int    `json:"target_id"`
	MappingType string `json:"mapping_type"`
	UsageCount  int    `json:"usage_count"`
}
```

Replace with:
```go
type MappingRow struct {
	MappingID       uint64  `json:"mapping_id"`
	OCRText         string  `json:"ocr_text"`
	TargetName      string  `json:"target_name"`
	TargetType      string  `json:"target_type"` // "product", "package", or "customer"
	TargetID        int     `json:"target_id"`
	MappingType     string  `json:"mapping_type"`
	ConfidenceScore float64 `json:"confidence_score"`
	UsageCount      int     `json:"usage_count"`
}
```

- [ ] **Step 2: Replace buildMappingRows with buildAllMappingRows**

Replace the existing `buildMappingRows` function (line ~3193) with:
```go
func buildAllMappingRows(
	productMappings []models.PDFProductMapping,
	packageMappings []models.PDFPackageMapping,
	customerMappings []models.PDFCustomerMapping,
	productNames map[int]string,
	packageNames map[int]string,
	customerNames map[int]string,
) []MappingRow {
	rows := make([]MappingRow, 0, len(productMappings)+len(packageMappings)+len(customerMappings))
	for _, m := range productMappings {
		name, ok := productNames[m.ProductID]
		if !ok { name = fmt.Sprintf("Product #%d", m.ProductID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFProductText, TargetName: name,
			TargetType: "product", TargetID: m.ProductID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	for _, m := range packageMappings {
		name, ok := packageNames[m.PackageID]
		if !ok { name = fmt.Sprintf("Package #%d", m.PackageID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFPackageText, TargetName: name,
			TargetType: "package", TargetID: m.PackageID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	for _, m := range customerMappings {
		name, ok := customerNames[m.CustomerID]
		if !ok { name = fmt.Sprintf("Customer #%d", m.CustomerID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFCustomerText, TargetName: name,
			TargetType: "customer", TargetID: m.CustomerID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	return rows
}
```

- [ ] **Step 3: Replace GetAllMappingsAPI to use new function + support ?type filter + customers**

Replace the entire `GetAllMappingsAPI` function (line ~3235):
```go
func (h *PDFHandler) GetAllMappingsAPI(c *gin.Context) {
	typeFilter := strings.ToLower(c.DefaultQuery("type", ""))

	var productMappings []models.PDFProductMapping
	var packageMappings []models.PDFPackageMapping
	var customerMappings []models.PDFCustomerMapping
	var err error

	if typeFilter == "" || typeFilter == "product" {
		productMappings, err = h.Mapper.GetAllMappings()
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load product mappings"}); return }
	}
	if typeFilter == "" || typeFilter == "package" {
		if h.PackageMapper != nil {
			packageMappings, err = h.PackageMapper.GetAllMappings()
			if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load package mappings"}); return }
		}
	}
	if typeFilter == "" || typeFilter == "customer" {
		if h.CustomerMapper != nil {
			if err := h.DB.Where("is_active = true").Order("usage_count DESC").Find(&customerMappings).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer mappings"}); return
			}
		}
	}

	productNames := make(map[int]string)
	if len(productMappings) > 0 {
		ids := make([]int, 0, len(productMappings))
		for _, m := range productMappings { ids = append(ids, m.ProductID) }
		var products []models.Product
		if err := h.DB.Select("productid, name").Where("productid IN ?", ids).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load product names"}); return
		}
		for _, p := range products { productNames[int(p.ProductID)] = p.Name }
	}

	packageNames := make(map[int]string)
	if len(packageMappings) > 0 {
		ids := make([]int, 0, len(packageMappings))
		for _, m := range packageMappings { ids = append(ids, m.PackageID) }
		var packages []models.ProductPackage
		if err := h.DB.Select("package_id, name").Where("package_id IN ?", ids).Find(&packages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load package names"}); return
		}
		for _, p := range packages { packageNames[p.PackageID] = p.Name }
	}

	customerNames := make(map[int]string)
	if len(customerMappings) > 0 {
		ids := make([]int, 0, len(customerMappings))
		for _, m := range customerMappings { ids = append(ids, m.CustomerID) }
		var customers []models.Customer
		if err := h.DB.Select("customer_id, first_name, last_name, company").Where("customer_id IN ?", ids).Find(&customers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer names"}); return
		}
		for _, cu := range customers {
			name := strings.TrimSpace(cu.FirstName + " " + cu.LastName)
			if cu.Company != nil && *cu.Company != "" { name = *cu.Company }
			customerNames[int(cu.CustomerID)] = name
		}
	}

	rows := buildAllMappingRows(productMappings, packageMappings, customerMappings, productNames, packageNames, customerNames)
	c.JSON(http.StatusOK, gin.H{"mappings": rows})
}
```

- [ ] **Step 4: Update ShowMappingManagement to use buildAllMappingRows**

Find the call to `buildMappingRows` inside `ShowMappingManagement` (line ~3477). Replace:
```go
rows := buildMappingRows(productMappings, packageMappings, productNames, packageNames)
```
With:
```go
var customerMappings []models.PDFCustomerMapping
if h.CustomerMapper != nil {
	h.DB.Where("is_active = true").Order("usage_count DESC").Find(&customerMappings)
}
customerIDSet := make(map[int]struct{})
for _, m := range customerMappings { customerIDSet[m.CustomerID] = struct{}{} }
customerNames := make(map[int]string)
if len(customerIDSet) > 0 {
	cids := make([]int, 0, len(customerIDSet))
	for id := range customerIDSet { cids = append(cids, id) }
	var customers []models.Customer
	if err := h.DB.Select("customer_id, first_name, last_name, company").Where("customer_id IN ?", cids).Find(&customers).Error; err == nil {
		for _, cu := range customers {
			name := strings.TrimSpace(cu.FirstName + " " + cu.LastName)
			if cu.Company != nil && *cu.Company != "" { name = *cu.Company }
			customerNames[int(cu.CustomerID)] = name
		}
	}
}
rows := buildAllMappingRows(productMappings, packageMappings, customerMappings, productNames, packageNames, customerNames)
```

Also check Customer model field names in `internal/models/models.go` — verify `FirstName`, `LastName`, `Company` match. Adjust if different.

- [ ] **Step 5: Build to verify**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```
Expected: no errors. If you see "buildMappingRows undefined", check for any remaining calls with:
```bash
grep -n buildMappingRows internal/handlers/pdf_handler.go
```

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/pdf_handler.go
git commit -m "feat: extend MappingRow with confidence, add customer mapping support to GetAllMappingsAPI"
```

---

## Task 2: Add 6 new API endpoints to pdf_handler.go

**Files:**
- Modify: `internal/handlers/pdf_handler.go` (append new functions at end of file)

- [ ] **Step 1: Add GetExtractionPreview**

Append to `internal/handlers/pdf_handler.go`:
```go
// GetExtractionPreview returns mapped items with resolved product names for preview step.
// GET /api/v1/pdf/extractions/:extraction_id/preview
func (h *PDFHandler) GetExtractionPreview(c *gin.Context) {
	extractionID := c.Param("extraction_id")
	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"}); return
	}
	var items []models.PDFExtractionItem
	h.DB.Where("extraction_id = ? AND (mapped_product_id IS NOT NULL OR mapped_package_id IS NOT NULL)", extractionID).Find(&items)

	type PreviewItem struct {
		ItemID     uint64  `json:"item_id"`
		Name       string  `json:"name"`
		RawText    string  `json:"raw_text"`
		Quantity   int     `json:"quantity"`
		UnitPrice  float64 `json:"unit_price"`
		LineTotal  float64 `json:"line_total"`
		TargetType string  `json:"target_type"`
		TargetID   int     `json:"target_id"`
	}

	productIDs, packageIDs := []int{}, []int{}
	for _, it := range items {
		if it.MappedProductID.Valid { productIDs = append(productIDs, int(it.MappedProductID.Int64)) }
		if it.MappedPackageID.Valid { packageIDs = append(packageIDs, int(it.MappedPackageID.Int64)) }
	}
	productNames := make(map[int]string)
	if len(productIDs) > 0 {
		var products []models.Product
		h.DB.Select("productid, name").Where("productid IN ?", productIDs).Find(&products)
		for _, p := range products { productNames[int(p.ProductID)] = p.Name }
	}
	packageNames := make(map[int]string)
	if len(packageIDs) > 0 {
		var packages []models.ProductPackage
		h.DB.Select("package_id, name").Where("package_id IN ?", packageIDs).Find(&packages)
		for _, p := range packages { packageNames[p.PackageID] = p.Name }
	}

	result := make([]PreviewItem, 0, len(items))
	for _, it := range items {
		qty := 1
		if it.Quantity.Valid { qty = int(it.Quantity.Int64) }
		up := 0.0
		if it.UnitPrice.Valid { up = it.UnitPrice.Float64 }
		lt := 0.0
		if it.LineTotal.Valid { lt = it.LineTotal.Float64 }
		pi := PreviewItem{ItemID: it.ItemID, RawText: it.RawProductText, Quantity: qty, UnitPrice: up, LineTotal: lt}
		if it.MappedProductID.Valid {
			pi.TargetType = "product"; pi.TargetID = int(it.MappedProductID.Int64)
			pi.Name = productNames[pi.TargetID]
			if pi.Name == "" { pi.Name = fmt.Sprintf("Produkt #%d", pi.TargetID) }
		} else if it.MappedPackageID.Valid {
			pi.TargetType = "package"; pi.TargetID = int(it.MappedPackageID.Int64)
			pi.Name = packageNames[pi.TargetID]
			if pi.Name == "" { pi.Name = fmt.Sprintf("Paket #%d", pi.TargetID) }
		}
		result = append(result, pi)
	}
	totalAmount := 0.0
	if extraction.TotalAmount.Valid { totalAmount = extraction.TotalAmount.Float64 }
	c.JSON(http.StatusOK, gin.H{"extraction_id": extraction.ExtractionID, "items": result, "total_amount": totalAmount, "item_count": len(result)})
}
```

- [ ] **Step 2: Add CreateMappingAPI**

```go
// CreateMappingAPI creates a standalone product mapping (no extraction context).
// POST /api/v1/pdf/mappings-create
// Body: { "pdf_text": "...", "product_id": 42 }
func (h *PDFHandler) CreateMappingAPI(c *gin.Context) {
	var req struct {
		PDFText   string `json:"pdf_text" binding:"required"`
		ProductID int    `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pdf_text and product_id are required"}); return
	}
	var product models.Product
	if err := h.DB.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"}); return
	}
	userID := int64(0)
	if uid, exists := c.Get("userid"); exists {
		if id, ok := uid.(int64); ok { userID = id }
	}
	if err := h.Mapper.SaveMapping(req.PDFText, req.ProductID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save mapping"}); return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "product_name": product.Name})
}
```

- [ ] **Step 3: Add UpdatePackageMappingAPI and DeletePackageMappingAPI**

```go
// PUT /api/v1/pdf/package-mappings/:id
// Body: { "package_id": 7 }
func (h *PDFHandler) UpdatePackageMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"}); return }
	var req struct{ PackageID int `json:"package_id" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package_id is required"}); return
	}
	var pkg models.ProductPackage
	if err := h.DB.First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"}); return
	}
	result := h.DB.Model(&models.PDFPackageMapping{}).Where("mapping_id = ?", mappingID).
		Updates(map[string]interface{}{"package_id": req.PackageID, "mapping_type": "manual", "is_active": true})
	if result.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"}); return }
	if result.RowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"}); return }
	c.JSON(http.StatusOK, gin.H{"success": true, "target_name": pkg.Name})
}

// DELETE /api/v1/pdf/package-mappings/:id
func (h *PDFHandler) DeletePackageMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"}); return }
	if h.PackageMapper == nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Package mapper not available"}); return }
	if err := h.PackageMapper.DeleteMapping(mappingID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { c.JSON(http.StatusNotFound, gin.H{"error": "Not found"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"}); return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 4: Add UpdateCustomerMappingAPI and DeleteCustomerMappingAPI**

```go
// PUT /api/v1/pdf/customer-mappings/:id
// Body: { "customer_id": 3 }
func (h *PDFHandler) UpdateCustomerMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"}); return }
	var req struct{ CustomerID int `json:"customer_id" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.CustomerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id is required"}); return
	}
	var customer models.Customer
	if err := h.DB.First(&customer, req.CustomerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"}); return
	}
	result := h.DB.Model(&models.PDFCustomerMapping{}).Where("mapping_id = ?", mappingID).
		Updates(map[string]interface{}{"customer_id": req.CustomerID, "mapping_type": "manual", "is_active": true})
	if result.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"}); return }
	if result.RowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"}); return }
	name := strings.TrimSpace(customer.FirstName + " " + customer.LastName)
	if customer.Company != nil && *customer.Company != "" { name = *customer.Company }
	c.JSON(http.StatusOK, gin.H{"success": true, "target_name": name})
}

// DELETE /api/v1/pdf/customer-mappings/:id
func (h *PDFHandler) DeleteCustomerMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"}); return }
	result := h.DB.Model(&models.PDFCustomerMapping{}).Where("mapping_id = ?", mappingID).Update("is_active", false)
	if result.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"}); return }
	if result.RowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"}); return }
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 5: Build**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/pdf_handler.go
git commit -m "feat: add GetExtractionPreview, CreateMappingAPI, and package/customer CRUD endpoints"
```

---

## Task 3: Register new routes in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add to /api/v1 group**

Find the `apiPDF := api.Group("/pdf")` block (line ~1466). After the existing routes, add:
```go
apiPDF.GET("/extractions/:extraction_id/preview", pdfHandler.GetExtractionPreview)
apiPDF.POST("/mappings-create", pdfHandler.CreateMappingAPI)
apiPDF.PUT("/package-mappings/:id", pdfHandler.UpdatePackageMappingAPI)
apiPDF.DELETE("/package-mappings/:id", pdfHandler.DeletePackageMappingAPI)
apiPDF.PUT("/customer-mappings/:id", pdfHandler.UpdateCustomerMappingAPI)
apiPDF.DELETE("/customer-mappings/:id", pdfHandler.DeleteCustomerMappingAPI)
```

Note: `POST /mappings-create` instead of `POST /mappings` to avoid a Gin route conflict with the existing `GET /mappings`.

- [ ] **Step 2: Add to legacy /api group**

Find `pdfAPI := legacyAPI.Group("/pdf")` (line ~1566). Add the same 6 routes replacing `apiPDF` with `pdfAPI`.

- [ ] **Step 3: Build**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: register new PDF mapping routes in main.go"
```

---

## Task 4: Create MappingModal.tsx

**Files:**
- Create: `web/src/components/MappingModal.tsx`

- [ ] **Step 1: Create the types + helper + InlineSearch sub-component**

Create `web/src/components/MappingModal.tsx` with:
```typescript
import { useState, useEffect, useRef, useCallback } from 'react';
import { X, CheckCircle, AlertCircle, Search, ChevronRight, ArrowLeft } from 'lucide-react';

// ── Types ──────────────────────────────────────────────────────────────────

interface NullInt64 { Int64: number; Valid: boolean; }
interface NullFloat64 { Float64: number; Valid: boolean; }

interface ExtractionItem {
  item_id: number;
  raw_product_text: string;
  quantity: NullInt64 | number | null;
  unit_price: NullFloat64 | number | null;
  line_total: NullFloat64 | number | null;
  mapped_product_id: NullInt64 | number | null;
  mapped_package_id: NullInt64 | number | null;
  mapping_status: 'pending' | 'auto_mapped' | 'user_confirmed' | 'user_rejected' | 'needs_creation';
  mapping_confidence: NullFloat64 | number | null;
}

interface PreviewItem {
  item_id: number;
  name: string;
  raw_text: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  target_type: 'product' | 'package';
  target_id: number;
}

interface SearchResult {
  id: number;
  name: string;
  type: 'product' | 'package';
  sub: string;
}

export interface MappedItem {
  product_id: number;
  name: string;
  quantity: number;
}

export interface MappingModalProps {
  uploadId: number;
  onComplete: (items: MappedItem[]) => void;
  onClose: () => void;
}

function getNullInt(v: NullInt64 | number | null | undefined): number {
  if (v == null) return 0;
  if (typeof v === 'number') return v;
  return v.Valid ? v.Int64 : 0;
}

function getNullFloat(v: NullFloat64 | number | null | undefined): number {
  if (v == null) return 0;
  if (typeof v === 'number') return v;
  return v.Valid ? v.Float64 : 0;
}

function isMapped(item: ExtractionItem): boolean {
  return (item.mapping_status === 'auto_mapped' || item.mapping_status === 'user_confirmed')
    && (getNullInt(item.mapped_product_id) > 0 || getNullInt(item.mapped_package_id) > 0);
}

// ── InlineSearch ────────────────────────────────────────────────────────────

function InlineSearch({ initialQuery, onSelect }: { initialQuery: string; onSelect: (r: SearchResult) => void }) {
  const [query, setQuery] = useState(initialQuery);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { inputRef.current?.focus(); }, []);

  const search = useCallback((q: string) => {
    if (q.trim().length < 2) { setResults([]); return; }
    setLoading(true);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(async () => {
      try {
        const [pRes, pkgRes] = await Promise.all([
          fetch(`/api/pdf/products/search?q=${encodeURIComponent(q)}&limit=5`, { credentials: 'include' }),
          fetch(`/api/pdf/packages/search?q=${encodeURIComponent(q)}&limit=3`, { credentials: 'include' }),
        ]);
        const pd = pRes.ok ? await pRes.json() : {};
        const pkd = pkgRes.ok ? await pkgRes.json() : {};
        const products: SearchResult[] = (pd.products || []).slice(0, 5).map((p: Record<string, unknown>) => ({
          id: (p.productID || p.ProductID) as number,
          name: (p.name || p.Name) as string,
          type: 'product' as const,
          sub: 'Produkt',
        }));
        const packages: SearchResult[] = (pkd.packages || []).slice(0, 3).map((p: Record<string, unknown>) => ({
          id: (p.package_id || p.PackageID) as number,
          name: (p.name || p.Name) as string,
          type: 'package' as const,
          sub: `Paket${p.package_code ? ' · ' + p.package_code : ''}`,
        }));
        setResults([...products, ...packages]);
      } finally { setLoading(false); }
    }, 300);
  }, []);

  return (
    <div className="relative">
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-500" />
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => { setQuery(e.target.value); search(e.target.value); }}
          placeholder="Produkt suchen…"
          className="w-full pl-8 pr-3 py-1.5 bg-black/30 border border-blue-500/60 rounded-lg text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-400"
        />
      </div>
      {(results.length > 0 || loading) && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-gray-900 border border-white/10 rounded-lg shadow-xl z-50 overflow-hidden">
          {loading && <div className="px-3 py-2 text-xs text-gray-500">Suche…</div>}
          {results.map((r) => (
            <button key={`${r.type}-${r.id}`} type="button" onClick={() => onSelect(r)}
              className="w-full flex items-center justify-between px-3 py-2 text-sm text-left hover:bg-white/5 transition-colors">
              <span className="text-white">{r.name}</span>
              <span className="text-xs text-gray-500 ml-2">{r.sub}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Append the main MappingModal component**

Append to `web/src/components/MappingModal.tsx`:
```typescript
// ── MappingModal ────────────────────────────────────────────────────────────

export default function MappingModal({ uploadId, onComplete, onClose }: MappingModalProps) {
  const [phase, setPhase] = useState<'loading' | 'mapping' | 'preview' | 'error'>('loading');
  const [extractionId, setExtractionId] = useState<number | null>(null);
  const [items, setItems] = useState<ExtractionItem[]>([]);
  const [activeSearch, setActiveSearch] = useState<number | null>(null);
  const [previewItems, setPreviewItems] = useState<PreviewItem[]>([]);
  const [totalAmount, setTotalAmount] = useState(0);
  const [errorMsg, setErrorMsg] = useState('');
  const [savingItem, setSavingItem] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        let extraction = null;
        for (let i = 0; i < 30; i++) {
          await new Promise((r) => setTimeout(r, 500));
          const res = await fetch(`/api/pdf/extraction/${uploadId}`, { credentials: 'include' });
          if (res.ok) { const d = await res.json(); if (d.extraction_id) { extraction = d; break; } }
        }
        if (!extraction) throw new Error('OCR-Timeout — bitte erneut versuchen');
        if (cancelled) return;
        setExtractionId(extraction.extraction_id);
        await fetch(`/api/pdf/auto-map/${extraction.extraction_id}`, { method: 'POST', credentials: 'include' });
        const res2 = await fetch(`/api/pdf/extraction/${uploadId}`, { credentials: 'include' });
        const final = await res2.json();
        if (cancelled) return;
        setItems(final.items || []);
        setPhase('mapping');
      } catch (e) {
        if (!cancelled) { setErrorMsg(e instanceof Error ? e.message : 'Fehler'); setPhase('error'); }
      }
    };
    load();
    return () => { cancelled = true; };
  }, [uploadId]);

  const mappedCount = items.filter(isMapped).length;
  const totalCount = items.length;
  const allMapped = mappedCount === totalCount && totalCount > 0;

  const handleSelect = async (item: ExtractionItem, result: SearchResult) => {
    setSavingItem(item.item_id);
    setActiveSearch(null);
    try {
      const body = result.type === 'package'
        ? { package_id: result.id, status: 'user_confirmed' }
        : { product_id: result.id, status: 'user_confirmed' };
      await fetch(`/api/pdf/items/${item.item_id}/mapping`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      setItems((prev) => prev.map((it) => it.item_id !== item.item_id ? it : {
        ...it,
        mapped_product_id: result.type === 'product' ? result.id : null,
        mapped_package_id: result.type === 'package' ? result.id : null,
        mapping_status: 'user_confirmed',
        mapping_confidence: 100,
      }));
    } finally { setSavingItem(null); }
  };

  const handleProceedToPreview = async () => {
    if (!extractionId) return;
    const res = await fetch(`/api/pdf/extractions/${extractionId}/preview`, { credentials: 'include' });
    const data = await res.json();
    setPreviewItems(data.items || []);
    setTotalAmount(data.total_amount || 0);
    setPhase('preview');
  };

  const handleConfirm = async () => {
    if (!extractionId) return;
    await fetch(`/api/pdf/extractions/${extractionId}/finalize`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      credentials: 'include', body: JSON.stringify({}),
    });
    const mapped: MappedItem[] = previewItems
      .filter((pi) => pi.target_type === 'product')
      .map((pi) => ({ product_id: pi.target_id, name: pi.name, quantity: pi.quantity }));
    onComplete(mapped);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-gray-900 border border-white/10 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
          <div className="flex items-center gap-3">
            {phase === 'preview' && (
              <button type="button" onClick={() => setPhase('mapping')} className="p-1 hover:bg-white/10 rounded-lg transition-colors">
                <ArrowLeft className="w-4 h-4" />
              </button>
            )}
            <h2 className="text-base font-semibold text-white">
              {phase === 'preview' ? 'Vorschau — Items zum Job hinzufügen' : 'PDF Mapping'}
            </h2>
          </div>
          <button type="button" onClick={onClose} className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {phase === 'loading' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
              <p className="text-sm text-gray-400">OCR läuft & Auto-Mapping…</p>
            </div>
          )}
          {phase === 'error' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-red-400">
              <AlertCircle className="w-8 h-8" /><p className="text-sm">{errorMsg}</p>
            </div>
          )}
          {phase === 'mapping' && (
            <>
              <div className="flex items-center gap-3 mb-5">
                <div className="flex-1 h-1.5 bg-white/10 rounded-full overflow-hidden">
                  <div className="h-full bg-blue-500 rounded-full transition-all duration-300"
                    style={{ width: totalCount > 0 ? `${(mappedCount / totalCount) * 100}%` : '0%' }} />
                </div>
                <span className="text-xs text-gray-400 whitespace-nowrap">{mappedCount}/{totalCount} gemappt</span>
              </div>
              <div className="space-y-2">
                {items.map((item) => {
                  const conf = getNullFloat(item.mapping_confidence);
                  const mapped = isMapped(item);
                  const isSearchOpen = activeSearch === item.item_id;
                  const saving = savingItem === item.item_id;
                  let rowBg = 'bg-orange-500/5 border border-orange-500/20';
                  if (item.mapping_status === 'user_confirmed') rowBg = 'bg-green-500/5 border border-green-500/20';
                  else if (mapped && conf >= 80) rowBg = 'bg-green-500/5 border border-green-500/10';
                  else if (mapped && conf >= 60) rowBg = 'bg-yellow-500/5 border border-yellow-500/20';

                  return (
                    <div key={item.item_id} className={`rounded-xl p-3 ${rowBg}`}>
                      <div className="grid grid-cols-[1fr_16px_1fr] gap-3 items-start">
                        <div>
                          <p className="text-sm text-gray-300">{item.raw_product_text}</p>
                          <p className="text-xs text-gray-600 mt-0.5">
                            {getNullInt(item.quantity)}× · {getNullFloat(item.unit_price).toFixed(2)} €
                          </p>
                        </div>
                        <span className="text-gray-600 text-sm text-center pt-1">→</span>
                        <div>
                          {saving ? (
                            <div className="flex items-center gap-2 py-1">
                              <div className="w-3.5 h-3.5 border border-blue-500 border-t-transparent rounded-full animate-spin" />
                              <span className="text-xs text-gray-500">Speichert…</span>
                            </div>
                          ) : mapped && !isSearchOpen ? (
                            <button type="button" onClick={() => setActiveSearch(item.item_id)}
                              className="flex items-center gap-2 group w-full text-left">
                              <CheckCircle className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-sm text-green-400 group-hover:text-green-300 transition-colors">
                                {item.mapping_status === 'user_confirmed' ? 'Manuell gemappt' : `Auto (${conf.toFixed(0)}%)`}
                              </span>
                              {conf > 0 && conf < 80 && (
                                <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                                  {conf.toFixed(0)}%
                                </span>
                              )}
                            </button>
                          ) : (
                            <InlineSearch
                              initialQuery={isSearchOpen ? '' : item.raw_product_text}
                              onSelect={(r) => handleSelect(item, r)}
                            />
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </>
          )}
          {phase === 'preview' && (
            <div className="space-y-3">
              <p className="text-sm text-gray-400">Diese Items werden zum Job hinzugefügt:</p>
              <div className="space-y-1">
                {previewItems.map((pi) => (
                  <div key={pi.item_id} className="flex items-center justify-between py-2 border-b border-white/5">
                    <div>
                      <span className="text-sm text-white">{pi.name}</span>
                      <span className="text-xs text-gray-500 ml-2">{pi.raw_text}</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-gray-400">
                      <span>{pi.quantity}×</span>
                      <span>{pi.unit_price.toFixed(2)} €</span>
                      <span className="font-medium text-gray-300">{pi.line_total.toFixed(2)} €</span>
                    </div>
                  </div>
                ))}
              </div>
              {totalAmount > 0 && (
                <div className="flex justify-end pt-2">
                  <span className="text-sm font-semibold text-white">Gesamt: {totalAmount.toFixed(2)} €</span>
                </div>
              )}
              <p className="text-xs text-gray-600 text-center pt-2">Mappings werden global gespeichert</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-white/10 flex items-center justify-between">
          {phase === 'mapping' && (
            <>
              <span className="text-xs text-gray-500">
                {!allMapped && `${totalCount - mappedCount} Item(s) noch nicht gemappt`}
              </span>
              <button type="button" onClick={handleProceedToPreview} disabled={!allMapped}
                className="flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-400 disabled:bg-gray-700 disabled:text-gray-500 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors">
                Weiter <ChevronRight className="w-4 h-4" />
              </button>
            </>
          )}
          {phase === 'preview' && (
            <>
              <button type="button" onClick={() => setPhase('mapping')} className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors">
                Zurück
              </button>
              <button type="button" onClick={handleConfirm}
                className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-500 text-white rounded-lg text-sm font-medium transition-colors">
                <CheckCircle className="w-4 h-4" /> Zum Job hinzufügen
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: TypeScript build check**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | head -40
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/MappingModal.tsx
git commit -m "feat: add MappingModal React component"
```

---

## Task 5: Integrate MappingModal into JobsPage.tsx

**Files:**
- Modify: `web/src/pages/JobsPage.tsx`

- [ ] **Step 1: Add import**

At the top of `JobsPage.tsx`, find the existing import block and add:
```typescript
import MappingModal, { type MappedItem } from '../components/MappingModal';
```

- [ ] **Step 2: Add pendingUploadId state to PdfImportBanner**

Inside `PdfImportBanner`, find:
```typescript
const [open, setOpen] = useState(false);
const [status, setStatus] = useState('');
const [error, setError] = useState('');
```
Add one more line:
```typescript
const [pendingUploadId, setPendingUploadId] = useState<number | null>(null);
```

- [ ] **Step 3: Replace the upload() function body**

Replace the body of the `upload` async function inside `PdfImportBanner`. The current body does upload → poll → auto-map → build products → `onApply`. Replace the entire body with:
```typescript
const file = fileRef.current?.files?.[0];
if (!file) { setError('Bitte eine PDF-Datei auswählen.'); return; }
setError(''); setStatus('Wird hochgeladen…');
const fd = new FormData();
fd.append('pdf', file);
try {
  const up = await fetch('/api/pdf/upload', { method: 'POST', body: fd, credentials: 'include' });
  const upData = await up.json();
  if (!up.ok || !upData.upload_id) throw new Error(upData.error || 'Upload fehlgeschlagen');
  setStatus('');
  setOpen(false);
  setPendingUploadId(upData.upload_id);
} catch (e) {
  setError(e instanceof Error ? e.message : 'Fehler beim Upload');
  setStatus('');
}
```

- [ ] **Step 4: Add MappingModal rendering in PdfImportBanner's return**

Inside `PdfImportBanner`'s return statement, before the closing `</>`, add:
```typescript
{pendingUploadId !== null && (
  <MappingModal
    uploadId={pendingUploadId}
    onComplete={(items: MappedItem[]) => {
      onApply({
        products: items.map((it) => ({
          product_id: it.product_id,
          name: it.name,
          quantity: it.quantity,
        })),
      });
      setPendingUploadId(null);
    }}
    onClose={() => {
      if (!window.confirm('Mapping nicht abgeschlossen — Items werden nicht hinzugefügt. Fortfahren?')) return;
      setPendingUploadId(null);
    }}
  />
)}
```

- [ ] **Step 5: Verify ProductSelection type compatibility**

Search for `ProductSelection` or `onApply` type in `JobsPage.tsx`:
```bash
grep -n "ProductSelection\|onApply" /opt/dev/cores/rentalcore/web/src/pages/JobsPage.tsx | head -10
```
Verify that `onApply` accepts `{ products: Array<{ product_id: number; name: string; quantity: number }> }`. If the type differs, adjust the mapping in Step 4 accordingly.

- [ ] **Step 6: Build**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | head -40
```

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/JobsPage.tsx
git commit -m "feat: replace silent PDF auto-map with MappingModal in JobsPage"
```

---

## Task 6: Redesign mapping_management.html

**Files:**
- Rewrite: `web/templates/mapping_management.html`

The template receives `{{.mappings}}` — a `[]MappingRow` slice where each row now has `TargetType` of `"product"`, `"package"`, or `"customer"` and a `ConfidenceScore` float.

The new design uses 3 tabs driven by JavaScript filtering of ALL_MAPPINGS (a JS array populated from the Go template). Edit/delete use different endpoints depending on `target_type`.

- [ ] **Step 1: Write the new template**

The complete new content for `web/templates/mapping_management.html` is long — write it in two parts using the Bash append pattern. First part (head + tabs + toolbar + new-form):

```bash
cat > web/templates/mapping_management.html << 'TMPL'
<!DOCTYPE html>
<html lang="de" data-theme="dark">
<head>
    <script>(function(){const t=localStorage.getItem("rc-theme")||"dark";document.documentElement.setAttribute("data-theme",t);})();</script>
    <meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mapping-Verwaltung - RentalCore</title>
    <link rel="manifest" href="/static/manifest.json">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="theme-color" content="#030712">
    <link rel="stylesheet" href="/static/css/rental-core-design.css?v={{.timestamp}}">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.2/font/bootstrap-icons.css" rel="stylesheet">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
</head>
<body class="rc-animate-fade-in">
{{template "navbar.html" .}}
<main class="rc-container rc-mt-lg">
  <div class="rc-page-header" style="margin-bottom:20px;">
    <div>
      <h1 class="rc-page-title"><i class="bi bi-database-gear"></i> Mapping-Verwaltung</h1>
      <p class="rc-text-muted">Gespeicherte Zuordnungen zwischen OCR-Texten und Produkten, Paketen und Kunden.</p>
    </div>
  </div>
  <div id="statusMsg"></div>

  <!-- Tabs -->
  <div style="display:flex;border-bottom:2px solid var(--rc-border);margin-bottom:20px;">
    <button class="tab-btn" id="tab-product" onclick="switchTab('product')"
      style="padding:10px 20px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-2px;font-size:14px;cursor:pointer;font-weight:500;">
      <i class="bi bi-tag"></i> Produkte (<span id="count-product">0</span>)
    </button>
    <button class="tab-btn" id="tab-package" onclick="switchTab('package')"
      style="padding:10px 20px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-2px;font-size:14px;cursor:pointer;">
      <i class="bi bi-box-seam"></i> Pakete (<span id="count-package">0</span>)
    </button>
    <button class="tab-btn" id="tab-customer" onclick="switchTab('customer')"
      style="padding:10px 20px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-2px;font-size:14px;cursor:pointer;">
      <i class="bi bi-person"></i> Kunden (<span id="count-customer">0</span>)
    </button>
  </div>

  <!-- Toolbar -->
  <div class="rc-card" style="margin-bottom:16px;padding:12px 16px;">
    <div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;">
      <input type="text" class="rc-input" id="searchInput" placeholder="OCR-Text oder Ziel suchen…"
             style="flex:1;min-width:180px;" oninput="renderTable()">
      <button type="button" class="rc-btn rc-btn-secondary rc-btn-sm" id="sortUnsureBtn"
              onclick="toggleSortUnsure()"><i class="bi bi-exclamation-triangle"></i> Unsichere zuerst</button>
      <button type="button" class="rc-btn rc-btn-sm" onclick="toggleNewForm()"
              style="background:var(--rc-primary);color:#fff;white-space:nowrap;"><i class="bi bi-plus"></i> Neu</button>
    </div>
  </div>

  <!-- New product mapping form -->
  <div id="newForm" class="rc-card" style="display:none;margin-bottom:16px;padding:16px;">
    <p style="font-size:12px;font-weight:600;color:var(--rc-text-secondary);margin-bottom:10px;text-transform:uppercase;letter-spacing:0.05em;">Neues Produkt-Mapping anlegen</p>
    <div style="display:grid;grid-template-columns:1fr 1fr auto;gap:10px;align-items:end;">
      <div>
        <label style="font-size:12px;color:var(--rc-text-secondary);display:block;margin-bottom:4px;">OCR-Text</label>
        <input type="text" class="rc-input" id="newOcrText" placeholder="z.B. Akku Ambientebeleuchtung">
      </div>
      <div style="position:relative;">
        <label style="font-size:12px;color:var(--rc-text-secondary);display:block;margin-bottom:4px;">Produkt</label>
        <input type="text" class="rc-input" id="newProductSearch" placeholder="Produkt suchen…" oninput="onNewProductSearch(this.value)">
        <div id="newProductResults" style="display:none;position:absolute;top:100%;left:0;right:0;z-index:200;background:var(--rc-bg-card);border:1px solid var(--rc-border);border-radius:6px;max-height:200px;overflow-y:auto;box-shadow:0 4px 12px rgba(0,0,0,0.3);"></div>
        <input type="hidden" id="newProductId">
      </div>
      <div style="display:flex;gap:6px;">
        <button type="button" class="rc-btn rc-btn-sm" onclick="submitNewMapping()" style="background:var(--rc-primary);color:#fff;"><i class="bi bi-check"></i> Anlegen</button>
        <button type="button" class="rc-btn rc-btn-sm rc-btn-secondary" onclick="toggleNewForm()"><i class="bi bi-x"></i></button>
      </div>
    </div>
  </div>

  <!-- Table -->
  <div class="rc-card">
    <div style="overflow-x:auto;">
      <table class="rc-table" style="width:100%;">
        <thead><tr>
          <th>OCR-Text</th><th>Ziel</th>
          <th style="width:140px;">Konfidenz</th><th style="width:80px;">Typ</th>
          <th style="width:70px;text-align:center;">Nutzung</th><th style="width:100px;text-align:right;">Aktionen</th>
        </tr></thead>
        <tbody id="mappingTbody"></tbody>
      </table>
      <div id="emptyState" style="display:none;text-align:center;padding:40px;color:var(--rc-text-secondary);">
        <i class="bi bi-database-slash" style="font-size:36px;"></i>
        <p style="margin-top:10px;font-size:14px;">Keine Mappings gefunden.</p>
      </div>
    </div>
  </div>
</main>
TMPL
```

- [ ] **Step 2: Append the JavaScript + data section**

```bash
cat >> web/templates/mapping_management.html << 'SCRIPTEOF'
<script>
var ALL_MAPPINGS=[{{range .mappings}}{mapping_id:{{.MappingID}},ocr_text:{{printf "%q" .OCRText}},target_name:{{printf "%q" .TargetName}},target_type:{{printf "%q" .TargetType}},target_id:{{.TargetID}},mapping_type:{{printf "%q" .MappingType}},confidence_score:{{.ConfidenceScore}},usage_count:{{.UsageCount}}},{{end}}];
var currentTab='product',sortUnsure=false,editTimers={};

function switchTab(t){
  currentTab=t;
  document.querySelectorAll('.tab-btn').forEach(function(b){b.style.color='var(--rc-text-secondary)';b.style.borderBottomColor='transparent';});
  var a=document.getElementById('tab-'+t);
  if(a){a.style.color='var(--rc-primary)';a.style.borderBottomColor='var(--rc-primary)';}
  renderTable();
}
function toggleSortUnsure(){
  sortUnsure=!sortUnsure;
  var b=document.getElementById('sortUnsureBtn');
  b.style.background=sortUnsure?'var(--rc-warning)':'';b.style.color=sortUnsure?'#000':'';
  renderTable();
}
function toggleNewForm(){var f=document.getElementById('newForm');f.style.display=f.style.display==='none'?'':'none';}
function matchQ(m,q){return !q||m.ocr_text.toLowerCase().includes(q)||m.target_name.toLowerCase().includes(q);}
function updateCounts(){
  var q=(document.getElementById('searchInput').value||'').toLowerCase();
  ['product','package','customer'].forEach(function(t){
    var el=document.getElementById('count-'+t);
    if(el)el.textContent=ALL_MAPPINGS.filter(function(m){return m.target_type===t&&matchQ(m,q);}).length;
  });
}
function confBar(s){
  var p=Math.round(s),c=p>=80?'var(--rc-success)':p>=60?'var(--rc-warning)':'var(--rc-danger)';
  return '<div style="display:flex;align-items:center;gap:6px;"><div style="flex:1;height:6px;background:var(--rc-bg-secondary);border-radius:3px;overflow:hidden;"><div style="width:'+p+'%;height:100%;background:'+c+';border-radius:3px;"></div></div><span style="font-size:11px;color:'+c+';">'+p+'%</span></div>';
}
function esc(s){return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
function renderTable(){
  var q=(document.getElementById('searchInput').value||'').toLowerCase();
  var rows=ALL_MAPPINGS.filter(function(m){return m.target_type===currentTab&&matchQ(m,q);});
  if(sortUnsure)rows.sort(function(a,b){return a.confidence_score-b.confidence_score;});
  updateCounts();
  var tb=document.getElementById('mappingTbody'),em=document.getElementById('emptyState');
  if(rows.length===0){tb.textContent='';em.style.display='';return;}
  em.style.display='none';
  var bc=currentTab==='package'?'rc-badge-info':currentTab==='customer'?'rc-badge-secondary':'rc-badge-success';
  var icon=currentTab==='package'?'<i class="bi bi-box-seam" style="color:var(--rc-warning);"></i>':currentTab==='customer'?'<i class="bi bi-person" style="color:var(--rc-info);"></i>':'<i class="bi bi-tag" style="color:var(--rc-success);"></i>';
  tb.textContent='';
  rows.forEach(function(m){
    var tc=m.mapping_type==='manual'?'rc-badge-success':m.mapping_type==='fuzzy'?'rc-badge-info':'rc-badge-secondary';
    var tr=document.createElement('tr');tr.dataset.id=m.mapping_id;
    tr.innerHTML='<td style="max-width:260px;word-break:break-word;"><span style="font-size:13px;">'+esc(m.ocr_text)+'</span></td>'
      +'<td>'
        +'<div id="view-'+m.mapping_id+'" style="display:flex;align-items:center;gap:6px;">'+icon+' <span id="vname-'+m.mapping_id+'" style="font-size:13px;">'+esc(m.target_name)+'</span></div>'
        +'<div id="edit-'+m.mapping_id+'" style="display:none;position:relative;">'
          +'<input class="rc-input rc-input-sm" style="width:100%;" placeholder="Suchen…" oninput="onEditSearch(this,'+m.mapping_id+',\''+m.target_type+'\')">'
          +'<div id="eres-'+m.mapping_id+'" style="display:none;position:absolute;top:100%;left:0;right:0;z-index:200;background:var(--rc-bg-card);border:1px solid var(--rc-border);border-radius:6px;max-height:180px;overflow-y:auto;box-shadow:0 4px 12px rgba(0,0,0,0.3);"></div>'
        +'</div>'
      +'</td>'
      +'<td>'+(m.confidence_score>0?confBar(m.confidence_score):'<span style="color:var(--rc-text-secondary);font-size:12px;">—</span>')+'</td>'
      +'<td><span class="rc-badge rc-badge-sm '+tc+'">'+esc(m.mapping_type)+'</span></td>'
      +'<td style="text-align:center;color:var(--rc-text-secondary);font-size:13px;">'+m.usage_count+'×</td>'
      +'<td style="text-align:right;">'
        +'<div style="display:flex;gap:4px;justify-content:flex-end;">'
          +'<button id="ebtn-'+m.mapping_id+'" class="rc-btn rc-btn-sm rc-btn-outline" onclick="startEdit('+m.mapping_id+')"><i class="bi bi-pencil"></i></button>'
          +'<button class="rc-btn rc-btn-sm rc-btn-danger" onclick="deleteMapping('+m.mapping_id+',\''+m.target_type+'\')"><i class="bi bi-trash"></i></button>'
        +'</div>'
      +'</td>';
    tb.appendChild(tr);
  });
}
function startEdit(id){
  document.getElementById('view-'+id).style.display='none';
  document.getElementById('edit-'+id).style.display='';
  document.getElementById('ebtn-'+id).style.display='none';
  document.getElementById('edit-'+id).querySelector('input').focus();
}
function cancelEdit(id){
  document.getElementById('view-'+id).style.display='';
  var e=document.getElementById('edit-'+id);e.style.display='none';e.querySelector('input').value='';
  document.getElementById('ebtn-'+id).style.display='';
  var r=document.getElementById('eres-'+id);r.style.display='none';r.textContent='';
}
function onEditSearch(inp,id,tt){
  var q=inp.value.trim(),re=document.getElementById('eres-'+id);
  if(q.length<2){re.style.display='none';return;}
  clearTimeout(editTimers[id]);
  editTimers[id]=setTimeout(function(){
    var ep=tt==='customer'
      ?[fetch('/api/pdf/customers/search?q='+encodeURIComponent(q)+'&limit=6').then(function(r){return r.ok?r.json():{};}).catch(function(){return {};})]
      :[fetch('/api/pdf/products/search?q='+encodeURIComponent(q)+'&limit=4').then(function(r){return r.ok?r.json():{};}).catch(function(){return {};}),
        fetch('/api/pdf/packages/search?q='+encodeURIComponent(q)+'&limit=3').then(function(r){return r.ok?r.json():{};}).catch(function(){return {};})];
    Promise.all(ep).then(function(res){
      var items=[];
      if(tt==='customer'){
        (res[0].customers||res[0]||[]).slice(0,6).forEach(function(c){
          var n=c.company||(((c.first_name||'')+' '+(c.last_name||'')).trim());
          items.push({id:c.customer_id||c.CustomerID,name:n,type:'customer',sub:'Kunde'});
        });
      }else{
        (res[0].products||[]).slice(0,4).forEach(function(p){items.push({id:p.productID||p.ProductID,name:p.name||p.Name,type:'product',sub:'Produkt'});});
        if(res[1])(res[1].packages||[]).slice(0,3).forEach(function(p){items.push({id:p.package_id||p.PackageID,name:p.name||p.Name,type:'package',sub:'Paket'});});
      }
      re.textContent='';
      if(items.length===0){var d=document.createElement('div');d.style.padding='8px 12px';d.style.fontSize='13px';d.style.color='var(--rc-text-secondary)';d.textContent='Keine Treffer';re.appendChild(d);}
      else items.forEach(function(item){
        var d=document.createElement('div');
        d.style.cssText='padding:8px 12px;cursor:pointer;display:flex;justify-content:space-between;align-items:center;font-size:13px;';
        d.addEventListener('mouseover',function(){d.style.background='var(--rc-bg-secondary)';});
        d.addEventListener('mouseout',function(){d.style.background='';});
        var ns=document.createElement('span');ns.textContent=item.name;
        var ss=document.createElement('span');ss.style.cssText='font-size:11px;color:var(--rc-text-secondary);';ss.textContent=item.sub;
        d.appendChild(ns);d.appendChild(ss);
        d.addEventListener('click',function(){confirmEdit(id,item.type,item.id,item.name);});
        re.appendChild(d);
      });
      re.style.display='';
    });
  },300);
}
function confirmEdit(id,tt,tid,tname){
  var url,body;
  if(tt==='product'){url='/api/v1/pdf/mappings/'+id;body=JSON.stringify({type:'product',target_id:tid});}
  else if(tt==='package'){url='/api/v1/pdf/package-mappings/'+id;body=JSON.stringify({package_id:tid});}
  else{url='/api/v1/pdf/customer-mappings/'+id;body=JSON.stringify({customer_id:tid});}
  fetch(url,{method:'PUT',headers:{'Content-Type':'application/json'},body:body})
    .then(function(r){return r.json().then(function(d){if(!r.ok||!d.success)throw new Error(d.error||'Fehler');return d;});})
    .then(function(){
      var m=ALL_MAPPINGS.find(function(m){return m.mapping_id===id;});
      if(m){m.target_name=tname;m.target_id=tid;m.mapping_type='manual';}
      cancelEdit(id);renderTable();showMsg('success','Mapping aktualisiert.');
    }).catch(function(e){showMsg('error',e.message);});
}
function deleteMapping(id,tt){
  if(!confirm('Mapping löschen?'))return;
  var url=tt==='package'?'/api/v1/pdf/package-mappings/'+id:tt==='customer'?'/api/v1/pdf/customer-mappings/'+id:'/api/v1/pdf/mappings/'+id+'?type=product';
  fetch(url,{method:'DELETE'}).then(function(r){return r.json().then(function(d){if(!r.ok||!d.success)throw new Error(d.error||'Fehler');});})
    .then(function(){ALL_MAPPINGS=ALL_MAPPINGS.filter(function(m){return m.mapping_id!==id;});renderTable();showMsg('success','Mapping gelöscht.');})
    .catch(function(e){showMsg('error',e.message);});
}
var newProdTimer;
function onNewProductSearch(q){
  var re=document.getElementById('newProductResults');
  document.getElementById('newProductId').value='';
  if(q.trim().length<2){re.style.display='none';return;}
  clearTimeout(newProdTimer);
  newProdTimer=setTimeout(function(){
    fetch('/api/pdf/products/search?q='+encodeURIComponent(q)+'&limit=6').then(function(r){return r.ok?r.json():{};})
      .then(function(data){
        var prods=data.products||[];
        re.textContent='';
        if(prods.length===0){var d=document.createElement('div');d.style.padding='8px 12px';d.style.fontSize='13px';d.style.color='var(--rc-text-secondary)';d.textContent='Keine Treffer';re.appendChild(d);}
        else prods.forEach(function(p){
          var d=document.createElement('div');d.style.cssText='padding:8px 12px;cursor:pointer;font-size:13px;';
          d.textContent=p.name||p.Name;
          d.addEventListener('mouseover',function(){d.style.background='var(--rc-bg-secondary)';});
          d.addEventListener('mouseout',function(){d.style.background='';});
          d.addEventListener('click',function(){
            document.getElementById('newProductSearch').value=p.name||p.Name;
            document.getElementById('newProductId').value=p.productID||p.ProductID;
            re.style.display='none';
          });
          re.appendChild(d);
        });
        re.style.display='';
      });
  },300);
}
function submitNewMapping(){
  var ocrText=(document.getElementById('newOcrText').value||'').trim();
  var pid=parseInt(document.getElementById('newProductId').value||'0',10);
  if(!ocrText){showMsg('error','OCR-Text ist erforderlich.');return;}
  if(!pid){showMsg('error','Bitte ein Produkt auswählen.');return;}
  fetch('/api/v1/pdf/mappings-create',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({pdf_text:ocrText,product_id:pid})})
    .then(function(r){return r.json().then(function(d){if(!r.ok||!d.success)throw new Error(d.error||'Fehler');return d;});})
    .then(function(d){
      showMsg('success','Mapping "'+d.product_name+'" angelegt.');
      document.getElementById('newOcrText').value='';
      document.getElementById('newProductSearch').value='';
      document.getElementById('newProductId').value='';
      document.getElementById('newForm').style.display='none';
      setTimeout(function(){window.location.reload();},800);
    }).catch(function(e){showMsg('error',e.message);});
}
function showMsg(type,msg){
  var el=document.getElementById('statusMsg');
  var d=document.createElement('div');
  d.className='rc-alert '+(type==='success'?'rc-alert-success':'rc-alert-danger');
  d.style.marginBottom='12px';
  d.textContent=(type==='success'?'✓ ':'✕ ')+msg;
  el.appendChild(d);
  setTimeout(function(){if(el.contains(d))el.removeChild(d);},4000);
}
switchTab('product');
</script>
</body></html>
SCRIPTEOF
```

- [ ] **Step 3: Verify template syntax**

```bash
cd /opt/dev/cores/rentalcore && go build ./... && echo "Build OK"
```
The Go template is compiled as part of the build. Check that `{{range .mappings}}` and `{{printf "%q" .OCRText}}` don't produce errors. If you see template errors at runtime, check the Go template syntax in the script block.

- [ ] **Step 4: Commit**

```bash
git add web/templates/mapping_management.html
git commit -m "feat: redesign mapping management page with 3 tabs, confidence bars, and new mapping form"
```

---

## Task 7: Full build and deploy

- [ ] **Step 1: Build React frontend**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build
```

- [ ] **Step 2: Build Go binary**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

- [ ] **Step 3: Check current Docker version from README**

```bash
grep -i "version\|docker\|nobentie" /opt/dev/cores/rentalcore/README.md | head -5
```

- [ ] **Step 4: Build and push Docker image** (replace X.Y with next version)

```bash
cd /opt/dev/cores/rentalcore
docker build -t nobentie/rentalcore:5.X.Y .
docker push nobentie/rentalcore:5.X.Y
docker tag nobentie/rentalcore:5.X.Y nobentie/rentalcore:latest
docker push nobentie/rentalcore:latest
```

- [ ] **Step 5: Push to GitHub**

```bash
cd /opt/dev/cores/rentalcore && git push origin main
```

- [ ] **Step 6: Update README version and push**

```bash
# Edit README.md to reflect new version number
git add README.md
git commit -m "docs: update version to 5.X.Y"
git push origin main
```

---

## Spec Coverage Checklist

| Requirement | Task |
|-------------|------|
| MappingModal opens after PDF upload | Task 4 + 5 |
| Progress bar (X/N gemappt) | Task 4 Step 2 |
| Auto-mapped items with confidence % | Task 4 Step 2 |
| Inline search dropdown (300ms debounce) | Task 4 Step 1 |
| PUT /api/pdf/items/:item_id/mapping on select | Task 4 Step 2 |
| Footer blocked until all items mapped | Task 4 Step 2 |
| Preview step before confirmation | Task 4 Step 2 |
| POST finalize on confirm | Task 4 Step 2 |
| Global mapping save via persistExtractionMappings | Handled by existing FinalizeExtraction |
| GET /api/pdf/extractions/:id/preview | Task 2 Step 1 |
| POST /api/pdf/mappings-create | Task 2 Step 2 |
| PUT/DELETE /api/pdf/package-mappings/:id | Task 2 Step 3 |
| PUT/DELETE /api/pdf/customer-mappings/:id | Task 2 Step 4 |
| 3 tabs in mapping_management.html | Task 6 |
| Confidence bar column | Task 6 |
| "Unsichere zuerst" sort | Task 6 |
| "+ Neu" form for product mappings | Task 6 |
| Customer mapping edit/delete | Task 6 |
| Warning on modal close without completion | Task 5 Step 4 |
