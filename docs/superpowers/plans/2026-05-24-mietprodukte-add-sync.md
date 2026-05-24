# Mietprodukte hinzufügen + Produktbedarf-Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mietprodukte können im Job-Positionen-Panel hinzugefügt werden (wie Produkte/Dienstleistungen), und jede Positions-Mutation synchronisiert automatisch `job_product_requirements` für WarehouseCore.

**Architecture:** Der `PositionHandler` bekommt Zugriff auf `requirementRepo` und `db`, um nach jeder Produkt-Positions-Mutation die `job_product_requirements`-Tabelle neu aufzubauen. Ein neuer `GET /api/v1/rental-catalog`-Endpunkt liefert aktive Mietartikel. Im Frontend wird die Mietprodukte-Sektion um einen Add-Button mit Katalog-Dropdown erweitert.

**Tech Stack:** Go 1.21, Gin, GORM, React 18, TypeScript, Tailwind CSS

**Next version:** v5.3.51

---

### Task 1: Backend — `syncRequirements` in PositionHandler

**Files:**
- Modify: `rentalcore/internal/handlers/position_handler.go:17-31, 69-132, 145-212`

- [ ] **Schritt 1: `PositionHandler` Struct und Konstruktor erweitern**

Ersetze in `position_handler.go` Zeilen 17–30:

```go
type PositionHandler struct {
	positionRepo    *repository.PositionRepository
	jobRepo         *repository.JobRepository
	requirementRepo *repository.RequirementRepository
	db              *gorm.DB
}

func NewPositionHandler(positionRepo *repository.PositionRepository, jobRepo *repository.JobRepository, requirementRepo *repository.RequirementRepository, db *gorm.DB) *PositionHandler {
	if err := ensureJobPriceColumns(db); err != nil {
		log.Printf("warning: failed to ensure job price columns: %v", err)
	}
	return &PositionHandler{
		positionRepo:    positionRepo,
		jobRepo:         jobRepo,
		requirementRepo: requirementRepo,
		db:              db,
	}
}
```

- [ ] **Schritt 2: `syncRequirements` Methode hinzufügen**

Füge direkt nach dem `ensureJobPriceColumns`-Block (nach Zeile ~36) ein:

```go
// syncRequirements rebuilds job_product_requirements from current product positions.
// Called after every position mutation so WarehouseCore stays in sync.
func (h *PositionHandler) syncRequirements(jobID uint) {
	var positions []models.JobPosition
	if err := h.db.Where("job_id = ? AND position_type = 'product'", jobID).Find(&positions).Error; err != nil {
		log.Printf("syncRequirements: query failed for job %d: %v", jobID, err)
		return
	}
	reqs := make([]models.JobProductRequirement, 0, len(positions))
	for _, pos := range positions {
		if pos.ProductID == nil {
			continue
		}
		qty := int(math.Round(pos.Quantity))
		if qty < 1 {
			qty = 1
		}
		reqs = append(reqs, models.JobProductRequirement{
			JobID:     jobID,
			ProductID: *pos.ProductID,
			Quantity:  qty,
		})
	}
	if err := h.requirementRepo.SaveRequirements(jobID, reqs); err != nil {
		log.Printf("syncRequirements: save failed for job %d: %v", jobID, err)
	}
}
```

- [ ] **Schritt 3: `syncRequirements` in `CreatePosition` aufrufen**

Am Ende von `CreatePosition`, direkt vor `c.JSON(http.StatusCreated, ...)` (nach Zeile ~131) einfügen:

```go
	if input.PositionType == "product" {
		h.syncRequirements(uint(jobID))
	}
```

- [ ] **Schritt 4: `syncRequirements` in `UpdatePosition` aufrufen**

In `UpdatePosition` gibt es am Ende ein `c.JSON(http.StatusOK, ...)`. Direkt davor einfügen (der jobID kommt aus `c.Param("id")`):

Zuerst oben in `UpdatePosition` jobID parsen — füge nach `posID`-Parsing hinzu:
```go
	jobID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
```

Dann vor dem abschließenden `c.JSON`:
```go
	if pos.PositionType == "product" {
		h.syncRequirements(uint(jobID))
	}
```

- [ ] **Schritt 5: `syncRequirements` in `DeletePosition` aufrufen**

In `DeletePosition` muss die Position vor dem Löschen gelesen werden (für jobID + type). Ersetze die aktuelle `DeletePosition`-Implementierung:

```go
func (h *PositionHandler) DeletePosition(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	pos, err := h.positionRepo.GetByID(uint(posID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	if err := h.positionRepo.Delete(uint(posID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if pos.PositionType == "product" {
		h.syncRequirements(pos.JobID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "position deleted"})
}
```

- [ ] **Schritt 6: Kompilieren**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Erwartung: kein Fehler. Bei Fehler `requirementRepo` oder `db` im Konstruktor vergessen?

- [ ] **Schritt 7: `main.go` — `NewPositionHandler`-Aufruf aktualisieren**

In `main.go` Zeile 469:
```go
// Alt:
positionHandler := handlers.NewPositionHandler(positionRepo, jobRepo, db.DB)
// Neu:
positionHandler := handlers.NewPositionHandler(positionRepo, jobRepo, requirementRepo, db.DB)
```

`requirementRepo` ist bereits auf Zeile 357 definiert.

- [ ] **Schritt 8: Erneut kompilieren**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Erwartung: keine Fehler.

- [ ] **Schritt 9: Commit**

```bash
cd /opt/dev/cores/rentalcore
git add internal/handlers/position_handler.go cmd/server/main.go
git commit -m "feat(positions): sync job_product_requirements after product position mutations"
```

---

### Task 2: Backend — `GetRentalCatalog`-Endpunkt

**Files:**
- Modify: `rentalcore/internal/handlers/position_handler.go` (neue Methode am Ende)
- Modify: `rentalcore/cmd/server/main.go` (Route registrieren)

- [ ] **Schritt 1: `GetRentalCatalog`-Methode in `position_handler.go` hinzufügen**

Am Ende der Datei (nach der letzten Funktion) einfügen:

```go
// GetRentalCatalog returns all active rental equipment items for selection in job positions.
func (h *PositionHandler) GetRentalCatalog(c *gin.Context) {
	var items []models.RentalEquipment
	if err := h.db.Where("is_active = ?", true).
		Order("product_name ASC").
		Select("equipment_id, product_name, supplier_name, rental_price, category").
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
```

- [ ] **Schritt 2: Route in `main.go` registrieren**

In `main.go` in der `setupRoutes`-Funktion, im `api`-Block (direkt neben den anderen `api.GET`-Routen, z.B. nach der `apiJobs`-Gruppe, ca. Zeile 1370):

```go
api.GET("/rental-catalog", positionHandler.GetRentalCatalog)
```

- [ ] **Schritt 3: Kompilieren und testen**

```bash
cd /opt/dev/cores/rentalcore && go build ./...
```

Manueller Schnelltest (optional, wenn lokaler Server läuft):
```bash
curl -s http://localhost:8081/api/v1/rental-catalog | jq '.items | length'
```
Erwartung: Zahl > 0 (Seeddaten aus Migration 021 vorhanden).

- [ ] **Schritt 4: Commit**

```bash
cd /opt/dev/cores/rentalcore
git add internal/handlers/position_handler.go cmd/server/main.go
git commit -m "feat(positions): add GET /rental-catalog endpoint for job position selection"
```

---

### Task 3: Frontend — `api.ts` erweitern

**Files:**
- Modify: `rentalcore/web/src/lib/api.ts:134-244`

- [ ] **Schritt 1: `RentalCatalogItem`-Interface hinzufügen**

In `api.ts` nach dem `JobTotals`-Interface (nach Zeile ~166) einfügen:

```typescript
export interface RentalCatalogItem {
  equipmentID: number;
  productName: string;
  supplierName: string;
  rentalPrice: number;
  category: string;
}
```

- [ ] **Schritt 2: `getRentalCatalog` zur `positionsApi` hinzufügen**

In `positionsApi` (ab Zeile 225) nach `updatePriceSettings` einfügen:

```typescript
  getRentalCatalog: () =>
    api.get<{ items: RentalCatalogItem[] }>('/rental-catalog'),
```

- [ ] **Schritt 3: TypeScript-Build prüfen**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | tail -20
```

Erwartung: keine Fehler.

- [ ] **Schritt 4: Commit**

```bash
cd /opt/dev/cores/rentalcore
git add web/src/lib/api.ts
git commit -m "feat(frontend): add RentalCatalogItem type and getRentalCatalog API method"
```

---

### Task 4: Frontend — `JobPositionsPanel.tsx` — Mietprodukte Add-UI

**Files:**
- Modify: `rentalcore/web/src/components/JobPositionsPanel.tsx`

- [ ] **Schritt 1: Import `RentalCatalogItem` und State hinzufügen**

Zeile 2 — Import erweitern:
```typescript
import { positionsApi, api } from '../lib/api';
import type { JobPosition, JobTotals, RentalCatalogItem } from '../lib/api';
```

State-Deklarationen (nach `services`-State, ca. Zeile 22):
```typescript
  const [rentalItems, setRentalItems] = useState<RentalCatalogItem[]>([]);
```

`adding`-Typ anpassen (Zeile ~18):
```typescript
  const [adding, setAdding] = useState<'product' | 'service' | 'rental' | null>(null);
```

- [ ] **Schritt 2: Rental-Katalog beim Mount laden**

Im bestehenden `useEffect` für Produkte/Dienstleistungen (ca. Zeile 44) den Rental-Fetch ergänzen:

```typescript
  useEffect(() => {
    api.get('/products', { params: { limit: 5000 } }).then(r => {
      const list = (r.data as any).products || r.data;
      if (Array.isArray(list)) setProducts(list);
    }).catch(() => {});
    api.get('/service-items').then(r => {
      const list = (r.data as any).service_items || (r.data as any).items || r.data;
      if (Array.isArray(list)) setServices(list);
    }).catch(() => {});
    positionsApi.getRentalCatalog().then(r => {
      setRentalItems(r.data.items || []);
    }).catch(() => {});
  }, []);
```

- [ ] **Schritt 3: `handleAdd` für `'rental'` erweitern**

Signatur ändern:
```typescript
  const handleAdd = async (type: 'product' | 'service' | 'rental', itemId: number) => {
```

Am Ende von `handleAdd`, nach dem `else`-Block für `service`:
```typescript
    } else {
      const item = rentalItems.find(r => r.equipmentID === itemId);
      if (!item) return;
      await positionsApi.create(jobId, {
        position_type: 'rental',
        rental_equipment_id: item.equipmentID,
        description: item.productName,
        quantity: 1,
        unit: 'Stück',
        unit_price: item.rentalPrice,
        follow_day_factor: 0,
      });
    }
```

- [ ] **Schritt 4: Mietprodukte-Sektion im JSX ersetzen**

Den aktuellen bedingten Block (ca. Zeile 168–178, `{rentalPositions.length > 0 && (...)}`) ersetzen durch eine immer angezeigte Sektion mit Add-Button:

```tsx
      {/* Mietprodukte Section */}
      <div className="glass-dark rounded-xl border border-white/10 p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Building2 className="w-4 h-4 text-accent-red" />
            <h3 className="font-semibold text-white">Mietprodukte ({rentalPositions.length})</h3>
          </div>
          <button
            onClick={() => setAdding(adding === 'rental' ? null : 'rental')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-red/10 text-accent-red hover:bg-accent-red/20 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Mietprodukt
          </button>
        </div>

        {adding === 'rental' && (
          <div className="mb-4 p-3 rounded-lg bg-white/[0.03] border border-white/10">
            <select
              className="w-full bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
              value=""
              onChange={e => { if (e.target.value) handleAdd('rental', parseInt(e.target.value)); }}
            >
              <option value="">Mietprodukt auswählen...</option>
              {rentalItems.map(r => (
                <option key={r.equipmentID} value={r.equipmentID}>
                  {r.productName} — {r.supplierName} ({fmt(r.rentalPrice)} €/Tag)
                </option>
              ))}
            </select>
          </div>
        )}

        {rentalPositions.length === 0 && adding !== 'rental' && (
          <p className="text-gray-500 text-sm py-2">Keine Mietprodukte hinzugefügt.</p>
        )}

        <div className="space-y-2">
          {rentalPositions.map(pos => (
            <PositionRow key={pos.position_id} pos={pos} onUpdate={handleUpdate} onDelete={handleDelete} showFactor={false} />
          ))}
        </div>
      </div>
```

- [ ] **Schritt 5: TypeScript-Build prüfen**

```bash
cd /opt/dev/cores/rentalcore/web && npm run build 2>&1 | tail -20
```

Erwartung: keine Fehler.

- [ ] **Schritt 6: Commit**

```bash
cd /opt/dev/cores/rentalcore
git add web/src/components/JobPositionsPanel.tsx
git commit -m "feat(frontend): add rental product selection to JobPositionsPanel"
```

---

### Task 5: Version bump, README, Docker, Deploy

**Files:**
- Modify: `rentalcore/README.md`

- [ ] **Schritt 1: README aktualisieren**

Im README den Versionsabschnitt anpassen — neuen Eintrag `v5.3.51` oben einfügen:

```markdown
### **v5.3.51** - feat: Mietprodukte zu Jobs hinzufügbar; Produktpositionen synchronisieren job_product_requirements für WarehouseCore-Packliste
```

- [ ] **Schritt 2: Alles pushen**

```bash
cd /opt/dev/cores/rentalcore
git add README.md
git commit -m "chore: bump version to v5.3.51"
git push
```

- [ ] **Schritt 3: Docker Image bauen**

```bash
cd /opt/dev/cores/rentalcore
docker build -t nobentie/rentalcore:5.3.51 .
```

Erwartung: Build erfolgreich, keine Fehler.

- [ ] **Schritt 4: Docker Images pushen**

```bash
docker push nobentie/rentalcore:5.3.51
docker tag nobentie/rentalcore:5.3.51 nobentie/rentalcore:latest
docker push nobentie/rentalcore:latest
```

- [ ] **Schritt 5: Deployment via Komodo**

```bash
curl -s -X POST https://komodo.server-nt.de/execute/DeployStack \
  -H "X-Api-Key: K-JjzIjQZH4Tb8VHbwsGI9jSPB3iVc7hA5xn4z3fe1" \
  -H "X-Api-Secret: S-LwBKLnHGEq1BemfiC3MafA8qecif1CpmgANlbBbn" \
  -H "Content-Type: application/json" \
  -d '{"stack": "cores", "services": ["rentalcore"]}'
```

Erwartung: HTTP 200 mit `"status": "Ok"` und einer Operation-ID.

- [ ] **Schritt 6: Deployment verifizieren**

```bash
ssh noah@docker03 "docker logs --tail=20 \$(docker ps --filter name=rentalcore --format '{{.Names}}' | head -1)"
```

Erwartung: Server gestartet, Version 5.3.51 in den Logs sichtbar.
