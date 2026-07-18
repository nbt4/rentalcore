# OCR Job Positions, Steuer, Datum & WarehouseCore Device-Swap — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** OCR-erstellte Jobs zeigen Positionen (Produkte, Mietprodukte, Dienstleistungen) mit editierbaren Preisen, Menge und Steuer direkt in der Job-Detailansicht; WarehouseCore tauscht beim Scan automatisch Geräte gleichen Produkttyps.

**Architecture:** Ansatz A — `FinalizeExtraction` ruft nach `assignProductsToJob` eine neue Funktion `createPositionsFromExtraction` auf, die `job_positions`-Einträge aus OCR-Items erzeugt. Schema-Erweiterung via Migration 040. Frontend-Änderungen in `MappingModal.tsx` (Datum-Inputs) und `JobPositionsPanel.tsx` (Steuer-Spalte, Gesamtpreis, Mietprodukt-Sektion). WarehouseCore erhält Device-Swap-Logik in `processOuttake`.

**Tech Stack:** Go 1.21 + GORM + Gin, React 18 + TypeScript, PostgreSQL 16

---

## File Map

**RentalCore — neue/geänderte Dateien:**
| Datei | Änderung |
|---|---|
| `migrations/040_job_positions_tax_rental.up.sql` | NEU — Spalten + Constraint |
| `migrations/postgresql/000_combined_init.sql` | MODIFY — job_positions Definition |
| `internal/models/job_position.go` | MODIFY — TaxRate, RentalEquipmentID |
| `internal/handlers/position_handler.go` | MODIFY — Create/Update Input + Handler |
| `internal/handlers/pdf_handler.go` | MODIFY — createPositionsFromExtraction |
| `web/src/lib/api.ts` | MODIFY — JobPosition Interface |
| `web/src/components/MappingModal.tsx` | MODIFY — editierbare Datum-Inputs |
| `web/src/components/JobPositionsPanel.tsx` | MODIFY — TaxRate, LineTotal, Rental-Sektion |

**WarehouseCore — neue/geänderte Dateien:**
| Datei | Änderung |
|---|---|
| `internal/models/scan.go` | MODIFY — ScanResponse + Swapped-Felder |
| `internal/services/scan_service.go` | MODIFY — processOuttake Device-Swap |

---

## Task 1: DB-Migration — job_positions Schema erweitern

**Files:**
- Create: `rentalcore/migrations/040_job_positions_tax_rental.up.sql`
- Modify: `cores/migrations/postgresql/000_combined_init.sql` (Zeile ~1074)

- [ ] **Schritt 1: Migration-Datei erstellen**

Inhalt von `rentalcore/migrations/040_job_positions_tax_rental.up.sql`:
```sql
-- 040_job_positions_tax_rental.up.sql
-- Add tax_rate per position (default 19%) and rental_equipment reference

ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS tax_rate DECIMAL(5,2) NOT NULL DEFAULT 19.00,
  ADD COLUMN IF NOT EXISTS rental_equipment_id INT REFERENCES rental_equipment(equipment_id) ON DELETE SET NULL;

ALTER TABLE job_positions
  DROP CONSTRAINT IF EXISTS job_positions_position_type_check;

ALTER TABLE job_positions
  ADD CONSTRAINT job_positions_position_type_check
  CHECK (position_type IN ('product', 'service', 'rental', 'package'));

CREATE INDEX IF NOT EXISTS idx_job_positions_rental_equipment ON job_positions(rental_equipment_id);
```

- [ ] **Schritt 2: combined_init.sql aktualisieren**

In `/opt/dev/cores/migrations/postgresql/000_combined_init.sql` die `job_positions` CREATE TABLE (Zeile ~1071) so anpassen:

```sql
CREATE TABLE IF NOT EXISTS job_positions (
    position_id     BIGSERIAL PRIMARY KEY,
    job_id          BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    position_type   VARCHAR(20) NOT NULL DEFAULT 'product' CHECK (position_type IN ('product', 'service', 'rental', 'package')),
    product_id      INT REFERENCES products(productid) ON DELETE SET NULL,
    service_item_id BIGINT REFERENCES service_items(id) ON DELETE SET NULL,
    rental_equipment_id INT REFERENCES rental_equipment(equipment_id) ON DELETE SET NULL,
    description     TEXT NOT NULL DEFAULT '',
    quantity        DECIMAL(10,2) NOT NULL DEFAULT 1,
    unit            VARCHAR(50) NOT NULL DEFAULT 'Stück',
    unit_price      DECIMAL(12,2) NOT NULL DEFAULT 0,
    follow_day_factor DECIMAL(4,2) NOT NULL DEFAULT 0.50,
    discount_percent  DECIMAL(5,2) NOT NULL DEFAULT 0,
    discount_amount   DECIMAL(12,2) NOT NULL DEFAULT 0,
    tax_rate          DECIMAL(5,2) NOT NULL DEFAULT 19.00,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Schritt 3: Migration auf Live-DB anwenden**

```bash
ssh noah@docker03
docker exec -i tscores-postgres-1 psql -U rentalcore -d rentalcore < /dev/stdin << 'EOF'
ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS tax_rate DECIMAL(5,2) NOT NULL DEFAULT 19.00,
  ADD COLUMN IF NOT EXISTS rental_equipment_id INT REFERENCES rental_equipment(equipment_id) ON DELETE SET NULL;

ALTER TABLE job_positions DROP CONSTRAINT IF EXISTS job_positions_position_type_check;
ALTER TABLE job_positions ADD CONSTRAINT job_positions_position_type_check
  CHECK (position_type IN ('product', 'service', 'rental', 'package'));

CREATE INDEX IF NOT EXISTS idx_job_positions_rental_equipment ON job_positions(rental_equipment_id);
EOF
```

Erwartete Ausgabe: `ALTER TABLE` / `CREATE INDEX`

- [ ] **Schritt 4: Commit**

```bash
cd /opt/dev/cores
git add rentalcore/migrations/040_job_positions_tax_rental.up.sql migrations/postgresql/000_combined_init.sql
git commit -m "feat(db): add tax_rate and rental_equipment_id to job_positions"
```

---

## Task 2: Go Model — JobPosition erweitern

**Files:**
- Modify: `rentalcore/internal/models/job_position.go`

- [ ] **Schritt 1: Felder und Association hinzufügen**

`internal/models/job_position.go` vollständig ersetzen:

```go
package models

import "time"

type JobPosition struct {
	PositionID         uint      `gorm:"primaryKey;column:position_id" json:"position_id"`
	JobID              uint      `gorm:"column:job_id;not null;index" json:"job_id"`
	PositionType       string    `gorm:"column:position_type;not null;default:product" json:"position_type"`
	ProductID          *uint     `gorm:"column:product_id" json:"product_id"`
	ServiceItemID      *uint     `gorm:"column:service_item_id" json:"service_item_id"`
	RentalEquipmentID  *uint     `gorm:"column:rental_equipment_id" json:"rental_equipment_id"`
	Description        string    `gorm:"column:description;not null;default:''" json:"description"`
	Quantity           float64   `gorm:"column:quantity;not null;default:1" json:"quantity"`
	Unit               string    `gorm:"column:unit;not null;default:Stück" json:"unit"`
	UnitPrice          float64   `gorm:"column:unit_price;not null;default:0" json:"unit_price"`
	FollowDayFactor    float64   `gorm:"column:follow_day_factor;not null;default:0.50" json:"follow_day_factor"`
	DiscountPercent    float64   `gorm:"column:discount_percent;not null;default:0" json:"discount_percent"`
	DiscountAmount     float64   `gorm:"column:discount_amount;not null;default:0" json:"discount_amount"`
	TaxRate            float64   `gorm:"column:tax_rate;not null;default:19.00" json:"tax_rate"`
	SortOrder          int       `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt          time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Product          *Product          `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
	ServiceItem      *ServiceItem      `gorm:"foreignKey:ServiceItemID;references:ID" json:"service_item,omitempty"`
	RentalEquipment  *RentalEquipment  `gorm:"foreignKey:RentalEquipmentID;references:EquipmentID" json:"rental_equipment,omitempty"`
	Devices          []JobPositionDevice `gorm:"foreignKey:PositionID;references:PositionID" json:"devices,omitempty"`
}

func (JobPosition) TableName() string { return "job_positions" }

type JobPositionDevice struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	PositionID uint      `gorm:"column:position_id;not null;index" json:"position_id"`
	DeviceID   string    `gorm:"column:device_id;not null" json:"device_id"`
	ScannedAt  time.Time `gorm:"column:scanned_at;default:CURRENT_TIMESTAMP" json:"scanned_at"`
	ScannedBy  string    `gorm:"column:scanned_by;default:''" json:"scanned_by"`
}

func (JobPositionDevice) TableName() string { return "job_position_devices" }

type ServiceItem struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	Name         string    `gorm:"column:name;not null" json:"name"`
	Description  string    `gorm:"column:description" json:"description"`
	DefaultPrice float64   `gorm:"column:default_price;default:0" json:"default_price"`
	Category     string    `gorm:"column:category" json:"category"`
	Unit         string    `gorm:"column:unit;default:pauschal" json:"unit"`
	IsActive     bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ServiceItem) TableName() string { return "service_items" }
```

- [ ] **Schritt 2: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Erwartete Ausgabe: kein Fehler

- [ ] **Schritt 3: Commit**

```bash
git add internal/models/job_position.go
git commit -m "feat(model): add TaxRate and RentalEquipmentID to JobPosition"
```

---

## Task 3: Go Backend — Position Handler erweitern

**Files:**
- Modify: `rentalcore/internal/handlers/position_handler.go` (Zeilen ~43–100, ~120–165)

- [ ] **Schritt 1: CreatePositionInput erweitern**

`CreatePositionInput` Struct (ca. Zeile 43) ersetzen:

```go
type CreatePositionInput struct {
	PositionType      string   `json:"position_type" binding:"required,oneof=product service rental package"`
	ProductID         *uint    `json:"product_id"`
	ServiceItemID     *uint    `json:"service_item_id"`
	RentalEquipmentID *uint    `json:"rental_equipment_id"`
	Description       string   `json:"description"`
	Quantity          float64  `json:"quantity"`
	Unit              string   `json:"unit"`
	UnitPrice         float64  `json:"unit_price"`
	FollowDayFactor   *float64 `json:"follow_day_factor"`
	DiscountPercent   float64  `json:"discount_percent"`
	DiscountAmount    float64  `json:"discount_amount"`
	TaxRate           *float64 `json:"tax_rate"`
}
```

- [ ] **Schritt 2: CreatePosition Handler — tax_rate + rental_equipment_id setzen**

Im `CreatePosition` Handler (nach `followDayFactor`-Block, vor `nextOrder`):

```go
// rental und package haben keinen follow_day_factor
if input.PositionType == "rental" || input.PositionType == "package" {
    followDayFactor = 0
}

taxRate := 19.0
if input.TaxRate != nil {
    taxRate = *input.TaxRate
}
```

Im `models.JobPosition{}` Block `TaxRate: taxRate` und `RentalEquipmentID: input.RentalEquipmentID` ergänzen:

```go
pos := models.JobPosition{
    JobID:             uint(jobID),
    PositionType:      input.PositionType,
    ProductID:         input.ProductID,
    ServiceItemID:     input.ServiceItemID,
    RentalEquipmentID: input.RentalEquipmentID,
    Description:       input.Description,
    Quantity:          input.Quantity,
    Unit:              input.Unit,
    UnitPrice:         input.UnitPrice,
    FollowDayFactor:   followDayFactor,
    DiscountPercent:   input.DiscountPercent,
    DiscountAmount:    input.DiscountAmount,
    TaxRate:           taxRate,
    SortOrder:         nextOrder,
}
```

- [ ] **Schritt 3: UpdatePositionInput — TaxRate ergänzen**

`UpdatePositionInput` Struct (ca. Zeile 115):

```go
type UpdatePositionInput struct {
	Description     *string  `json:"description"`
	Quantity        *float64 `json:"quantity"`
	Unit            *string  `json:"unit"`
	UnitPrice       *float64 `json:"unit_price"`
	FollowDayFactor *float64 `json:"follow_day_factor"`
	DiscountPercent *float64 `json:"discount_percent"`
	DiscountAmount  *float64 `json:"discount_amount"`
	TaxRate         *float64 `json:"tax_rate"`
}
```

Im `UpdatePosition` Handler nach dem `DiscountAmount`-Block einfügen:

```go
if input.TaxRate != nil {
    pos.TaxRate = *input.TaxRate
}
```

- [ ] **Schritt 4: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Erwartete Ausgabe: kein Fehler

- [ ] **Schritt 5: Commit**

```bash
git add internal/handlers/position_handler.go
git commit -m "feat(api): extend position create/update with tax_rate and rental type"
```

---

## Task 4: Go Backend — createPositionsFromExtraction

**Files:**
- Modify: `rentalcore/internal/handlers/pdf_handler.go`

- [ ] **Schritt 1: Neue Funktion einfügen**

Direkt nach der `assignProductsToJob`-Funktion (ca. Zeile 2256) einfügen:

```go
// createPositionsFromExtraction creates job_positions from PDF extraction items.
// Called after assignProductsToJob during finalize. Idempotent: deletes existing positions first.
func (h *PDFHandler) createPositionsFromExtraction(job *models.Job, extractionID uint64) error {
	var items []models.PDFExtractionItem
	if err := h.DB.Where(
		"extraction_id = ? AND mapping_status IN ('auto_mapped','user_confirmed')",
		extractionID,
	).Order("item_id ASC").Find(&items).Error; err != nil {
		return err
	}

	// Idempotent: remove existing positions for this job before recreating
	if err := h.DB.Where("job_id = ?", job.JobID).Delete(&models.JobPosition{}).Error; err != nil {
		return err
	}

	for i, item := range items {
		qty := 1.0
		if item.Quantity.Valid && item.Quantity.Int64 > 0 {
			qty = float64(item.Quantity.Int64)
		}
		unitPrice := 0.0
		if item.UnitPrice.Valid {
			unitPrice = item.UnitPrice.Float64
		}

		var posType string
		var productID *uint
		var serviceItemID *uint
		var rentalEquipmentID *uint
		followDayFactor := 0.5

		switch {
		case item.MappedProductID.Valid:
			posType = "product"
			pid := uint(item.MappedProductID.Int64)
			productID = &pid
		case item.MappedPackageID.Valid:
			posType = "package"
			followDayFactor = 0
		case item.MappedRentalEquipmentID.Valid:
			posType = "rental"
			rid := uint(item.MappedRentalEquipmentID.Int64)
			rentalEquipmentID = &rid
			followDayFactor = 0
		case item.MappedServiceItemID.Valid:
			posType = "service"
			sid := uint(item.MappedServiceItemID.Int64)
			serviceItemID = &sid
			followDayFactor = 0
		default:
			continue
		}

		pos := models.JobPosition{
			JobID:             uint(job.JobID),
			PositionType:      posType,
			ProductID:         productID,
			ServiceItemID:     serviceItemID,
			RentalEquipmentID: rentalEquipmentID,
			Description:       item.RawProductText,
			Quantity:          qty,
			Unit:              "Stück",
			UnitPrice:         unitPrice,
			FollowDayFactor:   followDayFactor,
			DiscountPercent:   0,
			TaxRate:           19.0,
			SortOrder:         i,
		}

		if err := h.DB.Create(&pos).Error; err != nil {
			log.Printf("[WARN] createPositionsFromExtraction: failed to create position for item %d: %v", item.ItemID, err)
		}
	}
	return nil
}
```

- [ ] **Schritt 2: Funktion in FinalizeExtraction aufrufen**

In `FinalizeExtraction`, an **beiden** Stellen wo `assignProductsToJob` aufgerufen wird (Zeilen ~1740 und ~1854), direkt danach einfügen:

```go
// Stelle 1 — bestehender Job (Re-Finalize), nach dem assignProductsToJob Block:
if posErr := h.createPositionsFromExtraction(&job, extraction.ExtractionID); posErr != nil {
    log.Printf("[WARN] createPositionsFromExtraction failed for job %d: %v", job.JobID, posErr)
}

// Stelle 2 — neuer Job, nach assignProductsToJob und vor h.DB.Model(...).Update("job_id"):
if posErr := h.createPositionsFromExtraction(&job, extraction.ExtractionID); posErr != nil {
    log.Printf("[WARN] createPositionsFromExtraction failed for job %d: %v", job.JobID, posErr)
}
```

- [ ] **Schritt 3: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Erwartete Ausgabe: kein Fehler

- [ ] **Schritt 4: Manuell testen**

Ein bestehendes OCR-Dokument nochmals finalisieren (Re-Finalize via Frontend oder direkt via API):

```bash
# Prüfen ob Positionen angelegt wurden (job_id durch echte ID ersetzen)
ssh noah@docker03
docker exec tscores-postgres-1 psql -U rentalcore -d rentalcore \
  -c "SELECT position_id, position_type, description, quantity, unit_price, tax_rate FROM job_positions WHERE job_id = <JOB_ID> ORDER BY sort_order;"
```

Erwartete Ausgabe: Rows mit den gemappten Produkten/Dienstleistungen des OCR-Dokuments

- [ ] **Schritt 5: Commit**

```bash
git add internal/handlers/pdf_handler.go
git commit -m "feat(ocr): create job_positions from extraction items on finalize"
```

---

## Task 5: Frontend — api.ts Typen aktualisieren

**Files:**
- Modify: `rentalcore/web/src/lib/api.ts` (Zeilen ~134–150)

- [ ] **Schritt 1: JobPosition Interface erweitern**

```typescript
export interface JobPosition {
  position_id: number;
  job_id: number;
  position_type: 'product' | 'service' | 'rental' | 'package';
  product_id: number | null;
  service_item_id: number | null;
  rental_equipment_id: number | null;
  description: string;
  quantity: number;
  unit: string;
  unit_price: number;
  follow_day_factor: number;
  discount_percent: number;
  discount_amount: number;
  tax_rate: number;
  sort_order: number;
  product?: { productID: number; name: string; itemcostperday?: number } | null;
  service_item?: { id: number; name: string; default_price?: number; unit?: string } | null;
  rental_equipment?: { equipmentID: number; productName: string; rentalPrice?: number } | null;
  devices?: JobPositionDevice[];
}
```

- [ ] **Schritt 2: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | tail -20
```

Erwartete Ausgabe: keine TypeScript-Fehler

- [ ] **Schritt 3: Commit**

```bash
git add web/src/lib/api.ts
git commit -m "feat(types): extend JobPosition with tax_rate, rental_equipment_id"
```

---

## Task 6: Frontend — MappingModal editierbare Datumsfelder

**Files:**
- Modify: `rentalcore/web/src/components/MappingModal.tsx` (Zeilen ~806–826)

- [ ] **Schritt 1: Meta-Bereich auf date-Inputs umbauen**

Den bestehenden Meta-Block (ca. Zeilen 806–826, `phase === 'mapping'` section) ersetzen:

```tsx
{/* Meta row: customer picker + dates */}
<div className="rc-card mb-4 px-3 py-2 flex flex-wrap gap-4 items-center text-xs" style={{ background: 'var(--rc-bg-secondary)' }}>
  {extractionId && (
    <CustomerPicker
      extractionId={extractionId}
      currentId={meta.customer_id}
      currentName={meta.customer_name}
      onChanged={(id, name) => setMeta(m => ({ ...m, customer_id: id, customer_name: name }))}
    />
  )}
  <label className="flex items-center gap-1.5" style={{ color: 'var(--rc-text-secondary)' }}>
    Von:
    <input
      type="date"
      value={meta.start_date ? meta.start_date.substring(0, 10) : ''}
      onChange={e => setMeta(m => ({ ...m, start_date: e.target.value }))}
      className="rc-input rc-input-sm"
      style={{ width: '130px', padding: '2px 6px' }}
    />
  </label>
  <label className="flex items-center gap-1.5" style={{ color: 'var(--rc-text-secondary)' }}>
    Bis:
    <input
      type="date"
      value={meta.end_date ? meta.end_date.substring(0, 10) : ''}
      onChange={e => setMeta(m => ({ ...m, end_date: e.target.value }))}
      className="rc-input rc-input-sm"
      style={{ width: '130px', padding: '2px 6px' }}
    />
  </label>
</div>
```

- [ ] **Schritt 2: handleConfirm — Datum mitsenden**

In `handleConfirm` (ca. Zeile 736) den Fetch-Body anpassen:

```typescript
const res = await fetch(`/api/pdf/extractions/${extractionId}/finalize`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({
    start_date: meta.start_date ? meta.start_date.substring(0, 10) : undefined,
    end_date: meta.end_date ? meta.end_date.substring(0, 10) : undefined,
  }),
});
```

- [ ] **Schritt 3: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | tail -20
```

- [ ] **Schritt 4: Commit**

```bash
git add web/src/components/MappingModal.tsx
git commit -m "feat(ocr): make start/end date editable in mapping modal"
```

---

## Task 7: Frontend — JobPositionsPanel — Steuer, Gesamtpreis, Mietprodukte

**Files:**
- Modify: `rentalcore/web/src/components/JobPositionsPanel.tsx`

- [ ] **Schritt 1: Import ergänzen**

Oben in der Datei den Import erweitern:

```typescript
import { Package, Wrench, Plus, Trash2, Check, Cpu, Building2 } from 'lucide-react';
```

- [ ] **Schritt 2: Mietprodukt-Positions filtern**

In `JobPositionsPanel` nach `servicePositions` einfügen:

```typescript
const rentalPositions = positions.filter(p => p.position_type === 'rental');
```

- [ ] **Schritt 3: Mietprodukt-Sektion nach der Dienstleistungs-Sektion hinzufügen**

Nach dem `{/* Services Section */}` Block (vor `{/* Totals */}`):

```tsx
{/* Rental Section */}
{rentalPositions.length > 0 && (
  <div className="glass-dark rounded-xl border border-white/10 p-5">
    <div className="flex items-center gap-2 mb-4">
      <Building2 className="w-4 h-4 text-accent-red" />
      <h3 className="font-semibold text-white">Mietprodukte ({rentalPositions.length})</h3>
    </div>
    <div className="space-y-2">
      {rentalPositions.map(pos => (
        <PositionRow key={pos.position_id} pos={pos} onUpdate={handleUpdate} onDelete={handleDelete} showFactor={false} />
      ))}
    </div>
  </div>
)}
```

- [ ] **Schritt 4: PositionRow — Gesamtpreis-Anzeige + tax_rate Spalte**

In `PositionRow` (ca. Zeile 246), nach dem `{/* Discount */}` Block und vor `{/* Delete */}`:

```tsx
{/* Tax Rate */}
<EditableCell
  value={pos.tax_rate}
  field="tax_rate"
  editing={editing}
  editVal={editVal}
  startEdit={startEdit}
  commitEdit={commitEdit}
  setEditVal={setEditVal}
  width="w-14"
  suffix="%"
/>

{/* Line Total (read-only) */}
<span className="w-20 text-xs text-right font-medium" style={{ color: 'var(--rc-text-primary)' }}>
  {fmt(pos.quantity * pos.unit_price * (1 - pos.discount_percent / 100))} €
</span>
```

- [ ] **Schritt 5: Build prüfen**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | tail -20
```

Erwartete Ausgabe: keine Fehler

- [ ] **Schritt 6: Commit**

```bash
git add web/src/components/JobPositionsPanel.tsx
git commit -m "feat(ui): add tax_rate, line total and rental positions to JobPositionsPanel"
```

---

## Task 8: RentalCore — Build und Deploy

- [ ] **Schritt 1: Versionsnummer ermitteln**

```bash
grep "^## \[" /opt/dev/cores/rentalcore/README.md | head -3
```

Notiere aktuelle Version (z.B. `5.3.33`) und erhöhe Patch um 1 (→ `5.3.34`).

- [ ] **Schritt 2: README aktualisieren**

In `README.md` neuen Eintrag unter der aktuellen Version hinzufügen (Version aus Schritt 1):

```markdown
## [5.3.34] - 2026-05-07
### Added
- OCR-Finalize erstellt job_positions aus Extraction-Items (Produkte, Mietprodukte, Dienstleistungen, Pakete)
- Steuer-Spalte (tax_rate, Standard 19%) pro Position editierbar
- Gesamtpreis pro Zeile in JobPositionsPanel
- Mietprodukt-Sektion in JobPositionsPanel
- Start/End-Datum im OCR Mapping-Modal editierbar
```

- [ ] **Schritt 3: GitHub pushen**

```bash
cd /opt/dev/cores/rentalcore
git add README.md
git commit -m "chore: bump version to 5.3.34"
git push origin main
```

- [ ] **Schritt 4: Docker Image bauen und pushen**

```bash
cd /opt/dev/cores/rentalcore
docker build -t nobentie/rentalcore:5.3.34 .
docker push nobentie/rentalcore:5.3.34
docker tag nobentie/rentalcore:5.3.34 nobentie/rentalcore:latest
docker push nobentie/rentalcore:latest
```

---

## Task 9: WarehouseCore — ScanResponse Modell erweitern

**Files:**
- Modify: `warehousecore/internal/models/scan.go` (Zeilen ~35–50)

- [ ] **Schritt 1: Felder zu ScanResponse hinzufügen**

In `ScanResponse` Struct (nach `SuggestedDeps`):

```go
type ScanResponse struct {
	Success        bool                          `json:"success"`
	Message        string                        `json:"message"`
	Device         *DeviceWithDetails             `json:"device,omitempty"`
	Product        *ProductInfo                   `json:"product,omitempty"`
	Movement       *DeviceMovement                `json:"movement,omitempty"`
	Action         string                         `json:"action"`
	PreviousStatus string                         `json:"previous_status,omitempty"`
	NewStatus      string                         `json:"new_status,omitempty"`
	Duplicate      bool                           `json:"duplicate"`
	JobInfo        *JobInfo                       `json:"job_info,omitempty"`
	SuggestedDeps  []ProductDependencyWithDetails `json:"suggested_dependencies,omitempty"`
	Swapped        bool                           `json:"swapped,omitempty"`
	SwappedFrom    string                         `json:"swapped_from,omitempty"`
}
```

- [ ] **Schritt 2: Build prüfen**

```bash
cd /opt/dev/cores/warehousecore && go build ./...
```

Erwartete Ausgabe: kein Fehler

- [ ] **Schritt 3: Commit**

```bash
cd /opt/dev/cores/warehousecore
git add internal/models/scan.go
git commit -m "feat(model): add Swapped fields to ScanResponse"
```

---

## Task 10: WarehouseCore — Device-Swap in processOuttake

**Files:**
- Modify: `warehousecore/internal/services/scan_service.go` (Zeilen ~177–290)

- [ ] **Schritt 1: Swap-Logik vor dem normalen Outtake einfügen**

In `processOuttake`, direkt nach dem `previousStatus`/`fromZoneID`-Block (nach Zeile ~185), **vor** dem `UPDATE devices SET status = 'on_job'`:

```go
// Device-Swap: If this device isn't already assigned to the job,
// check if another device of the same product type is pending — if so, swap it.
swapped := false
swappedFrom := ""

if jobID != nil {
	// Check if this device is already in the job
	var alreadyAssigned int
	tx.QueryRow(`
		SELECT COUNT(*) FROM job_devices WHERE deviceID = $1 AND jobID = $2
	`, device.DeviceID, *jobID).Scan(&alreadyAssigned)

	if alreadyAssigned == 0 {
		// Find a pending device of the same product in this job
		var candidateDeviceID string
		err := tx.QueryRow(`
			SELECT jd.deviceID
			FROM job_devices jd
			JOIN devices d ON d.deviceID = jd.deviceID
			WHERE jd.jobID = $1
			  AND d.productID = $2
			  AND jd.pack_status = 'pending'
			  AND jd.deviceID != $3
			LIMIT 1
		`, *jobID, device.ProductID, device.DeviceID).Scan(&candidateDeviceID)

		if err == nil && candidateDeviceID != "" {
			// Remove the old device from job_devices
			_, err = tx.Exec(`
				DELETE FROM job_devices WHERE deviceID = $1 AND jobID = $2
			`, candidateDeviceID, *jobID)
			if err != nil {
				log.Printf("[WARN] device swap: failed to remove old device %s: %v", candidateDeviceID, err)
			} else {
				swapped = true
				swappedFrom = candidateDeviceID
				log.Printf("[INFO] device swap: replaced %s with %s for job %d", candidateDeviceID, device.DeviceID, *jobID)
			}
		}
	}
}
```

- [ ] **Schritt 2: Swap-Ergebnis in Response zurückgeben**

Am Ende von `processOuttake`, die `return`-Anweisung anpassen:

```go
return &models.ScanResponse{
	Success:        true,
	Message:        "Device assigned to job",
	Action:         "outtake",
	PreviousStatus: previousStatus,
	NewStatus:      "on_job",
	SuggestedDeps:  suggestedDeps,
	Swapped:        swapped,
	SwappedFrom:    swappedFrom,
}, movement, nil
```

- [ ] **Schritt 3: Build prüfen**

```bash
cd /opt/dev/cores/warehousecore && go build ./...
```

Erwartete Ausgabe: kein Fehler

- [ ] **Schritt 4: Manuell testen**

```bash
# Einen Job mit 2 gleichen Geräten in job_devices (pack_status='pending') anlegen
# Dann ein drittes Gerät desselben Produkts einscannen
# Erwartung: Response enthält "swapped": true, "swapped_from": "<altes Gerät>"
ssh noah@docker03
docker logs tscores-warehousecore-1 --tail=20
# Erwartete Log-Zeile: "[INFO] device swap: replaced TOP1001 with TOP1002 for job 42"
```

- [ ] **Schritt 5: Commit**

```bash
cd /opt/dev/cores/warehousecore
git add internal/services/scan_service.go
git commit -m "feat(scan): swap pending device on outtake if same product already assigned"
```

---

## Task 11: WarehouseCore — Build und Deploy

- [ ] **Schritt 1: Versionsnummer ermitteln**

```bash
grep "^## \[" /opt/dev/cores/warehousecore/README.md | head -3
```

Notiere aktuelle Version und erhöhe Patch um 1.

- [ ] **Schritt 2: README aktualisieren**

```markdown
## [X.Y.Z] - 2026-05-07
### Added
- Device-Swap beim Scan: Gerät gleichen Produkttyps mit pack_status='pending' wird automatisch ersetzt
- ScanResponse enthält swapped/swapped_from Felder zur UI-Rückmeldung
```

- [ ] **Schritt 3: GitHub pushen**

```bash
cd /opt/dev/cores/warehousecore
git add README.md
git commit -m "chore: bump version"
git push origin main
```

- [ ] **Schritt 4: Docker Image bauen und pushen**

```bash
cd /opt/dev/cores/warehousecore
# Version aus Schritt 1 einsetzen
docker build -t nobentie/warehousecore:X.Y.Z .
docker push nobentie/warehousecore:X.Y.Z
docker tag nobentie/warehousecore:X.Y.Z nobentie/warehousecore:latest
docker push nobentie/warehousecore:latest
```

---

## Reihenfolge der Tasks

```
Task 1 (DB) → Task 2 (Model) → Task 3 (Handler) → Task 4 (PDF Finalize)
                                                  ↓
Task 5 (API Types) → Task 6 (MappingModal) → Task 7 (PositionsPanel)
                                                  ↓
                                            Task 8 (Deploy RC)

Task 9 (WC Model) → Task 10 (WC Swap) → Task 11 (Deploy WC)
```

Tasks 1–8 (RentalCore) und Tasks 9–11 (WarehouseCore) sind **voneinander unabhängig** und können parallel ausgeführt werden.
