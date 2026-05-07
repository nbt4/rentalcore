# Design: OCR Job Positions, Steuer, Datum & WarehouseCore Device-Swap

**Datum:** 2026-05-07  
**Status:** Genehmigt

## Überblick

Wenn ein Job via OCR erstellt wird, ist das Positionen-Panel in der Job-Detailansicht aktuell leer — `assignProductsToJob` erzeugt nur `job_devices`, aber keine `job_positions`. Dieses Feature füllt diese Lücke und fügt weitere Verbesserungen hinzu.

## Scope

Fünf zusammenhängende Änderungen:

1. DB-Schema: `tax_rate` und `rental_equipment_id` in `job_positions`
2. Backend: OCR-Finalize erzeugt `job_positions` aus Extraction-Items
3. MappingModal: Start/End-Datum editierbar
4. JobPositionsPanel: Steuer-Spalte, Gesamtpreis pro Zeile, Mietprodukt-Sektion
5. WarehouseCore: Device-Swap beim Scan (gleiche ProductID)

---

## 1. Datenbankschema

**Dateien:** `migrations/postgresql/001_rentalcore_schema.sql`, neue Migration

```sql
ALTER TABLE job_positions
  ADD COLUMN tax_rate DECIMAL(5,2) NOT NULL DEFAULT 19.00,
  ADD COLUMN rental_equipment_id INT REFERENCES rental_equipment(id);
```

- `tax_rate`: Standard 19%, pro Position überschreibbar
- `rental_equipment_id`: NULL für product/service, gesetzt für rental-Positionen
- `position_type` Constraint wird um `'rental'` und `'package'` erweitert

---

## 2. Backend — Positionen aus OCR erzeugen

**Datei:** `internal/handlers/pdf_handler.go`

### Neue Funktion `createPositionsFromExtraction(job *models.Job, extractionID uint64) error`

Wird in `FinalizeExtraction` direkt nach `assignProductsToJob` aufgerufen.

**Logik:**
1. Alle gemappten Items der Extraction laden (`mapping_status IN ('auto_mapped', 'user_confirmed')`)
2. Existierende Positionen für den Job löschen (idempotent bei Re-Finalize)
3. Für jedes Item eine `job_position` anlegen:

| `mapped_*_id` vorhanden | `position_type` | Pflichtfelder |
|---|---|---|
| `mapped_product_id` | `product` | `product_id`, `description`, `quantity`, `unit_price` |
| `mapped_package_id` | `package` | `description`, `quantity`, `unit_price` |
| `mapped_rental_equipment_id` | `rental` | `rental_equipment_id`, `description`, `quantity`, `unit_price` |
| `mapped_service_item_id` | `service` | `service_item_id`, `description`, `quantity`, `unit_price` |

- `quantity` und `unit_price` aus den OCR-Extraction-Items
- `tax_rate = 19.0` für alle Positionen
- `follow_day_factor = 0.5` für `product`, `0.0` für alle anderen
- `discount_percent = 0.0`
- `sort_order` = Reihenfolge der Items in der Extraction

### Erweiterung `CreatePositionInput`

```go
type CreatePositionInput struct {
    PositionType        string   `json:"position_type" binding:"required,oneof=product service rental package"`
    ProductID           *uint    `json:"product_id"`
    ServiceItemID       *uint    `json:"service_item_id"`
    RentalEquipmentID   *uint    `json:"rental_equipment_id"`
    Description         string   `json:"description"`
    Quantity            float64  `json:"quantity"`
    Unit                string   `json:"unit"`
    UnitPrice           float64  `json:"unit_price"`
    FollowDayFactor     float64  `json:"follow_day_factor"`
    DiscountPercent     float64  `json:"discount_percent"`
    TaxRate             float64  `json:"tax_rate"`
}
```

### Erweiterung `UpdatePosition`

`tax_rate` als patchbares Feld aufnehmen.

---

## 3. MappingModal — editierbare Datumsfelder

**Datei:** `web/src/components/MappingModal.tsx`

Im `phase === 'mapping'` Meta-Bereich (aktuell Zeile ~806):

- `start_date` und `end_date` werden von schreibgeschütztem Text zu `<input type="date">` umgebaut
- Änderungen aktualisieren nur den lokalen `meta`-State
- Beim Finalize werden `start_date` und `end_date` aus dem State an den Endpoint geschickt (der Endpoint akzeptiert diese bereits)
- Format: `YYYY-MM-DD` (HTML date input nativ)
- Wenn kein Datum aus OCR erkannt: leerer Input (kein Pflichtfeld)

---

## 4. JobPositionsPanel — Erweiterungen

**Datei:** `web/src/components/JobPositionsPanel.tsx`

### 4a. `PositionRow` — neue Spalte `tax_rate`

- Editierbar via `EditableCell`, Suffix `%`, Default 19
- Wird nach `discount_percent` eingereiht
- API-Call `PATCH /jobs/:id/positions/:posId` mit `{ tax_rate: value }`

### 4b. Gesamtpreis pro Zeile (read-only)

Berechnung im Frontend:
```ts
const lineTotal = pos.quantity * pos.unit_price * (1 - pos.discount_percent / 100);
```
Angezeigt als fetter Wert ganz rechts in der Zeile (vor dem Löschen-Button).

### 4c. Neuer Abschnitt „Mietprodukte"

- Filtert `positions.filter(p => p.position_type === 'rental')`
- Icon: `Building2` (bereits importiert in MappingModal, muss in Panel importiert werden)
- `PositionRow` mit `showFactor={false}` (kein `follow_day_factor` für Mietprodukte)
- Kein „Hinzufügen"-Button vorerst (werden nur via OCR befüllt)

### 4d. API-Typen

`JobPosition` Interface in `lib/api.ts` wird um `tax_rate: number` und `rental_equipment_id?: number` erweitert.

---

## 5. WarehouseCore — Device-Swap beim Scan

**Datei:** `warehousecore/internal/services/scan_service.go`

### Erweiterung `processOuttake`

Neue Logik **vor** dem normalen Insert in `job_devices`:

```
1. Prüfe: Ist deviceID bereits in job_devices für diesen Job?
   → Ja: normaler Duplicate-Check (bestehend)
   
2. Prüfe: Hat das gescannte Gerät eine productID X?
   → Gibt es in job_devices für diesen Job ein anderes Gerät
     mit derselben productID X und pack_status = 'pending'?
   
3. Wenn Treffer gefunden (swap_candidate):
   → swap_candidate aus job_devices entfernen (DELETE)
   → Neues Gerät einfügen mit pack_status = 'issued'
   → Response: { success: true, swapped: true, swapped_from: "TOP1001" }
   
4. Wenn kein Treffer:
   → Normaler Outtake-Flow (bestehend)
```

Die Prüfung läuft innerhalb derselben Transaktion.

### Erweiterung `ScanResponse`

```go
type ScanResponse struct {
    // ... bestehende Felder ...
    Swapped     bool   `json:"swapped,omitempty"`
    SwappedFrom string `json:"swapped_from,omitempty"`
}
```

Das WarehouseCore-Frontend zeigt bei `swapped: true` eine Info-Meldung: `"TOP1001 → TOP1002 getauscht"`.

---

## Nicht im Scope

- Mietprodukte manuell zur Positionsliste hinzufügen (nur via OCR)
- Pakete in eigener Sektion im Panel (werden als generische Position angezeigt)
- Gemischte Steuersätze im Totals-Panel aufschlüsseln (Gesamtsteuerbetrag bleibt)
- WarehouseCore-Frontend-Änderungen über die Swap-Meldung hinaus

---

## Betroffene Dateien

**RentalCore:**
- `migrations/postgresql/001_rentalcore_schema.sql` + neue Migration
- `internal/models/job_position.go`
- `internal/handlers/pdf_handler.go`
- `internal/handlers/position_handler.go`
- `web/src/components/MappingModal.tsx`
- `web/src/components/JobPositionsPanel.tsx`
- `web/src/lib/api.ts`

**WarehouseCore:**
- `internal/services/scan_service.go`
- `internal/models/scan.go`
