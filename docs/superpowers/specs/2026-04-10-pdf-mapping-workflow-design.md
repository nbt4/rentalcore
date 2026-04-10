# PDF Mapping Workflow — Design Spec
**Datum:** 2026-04-10  
**Status:** Approved  
**Bereich:** RentalCore — PDF-Import & Produkt-Mapping

---

## Überblick

Wenn ein Job erstellt wird, kann eine PDF (Angebot/Rechnung) hochgeladen werden. Diese wird geparst, erkannte Items werden automatisch auf Produkte in der DB gemappt. Ein Modal führt den Nutzer durch das Mapping — alle Items müssen gemappt sein bevor der Job-Import abgeschlossen wird. Jedes bestätigte Mapping wird global gespeichert, sodass zukünftige PDFs automatisch erkannt werden. Zusätzlich gibt es eine separate Management-Seite für alle Mappings.

---

## User Flow

```
PDF hochladen → parsen (async) → Mapping-Modal öffnet sich
→ Auto-gemappte Items grün (Konfidenz %)
→ Nicht-gemappte Items: Inline-Suche (Dropdown)
→ Fortschrittsbalken (X/N gemappt)
→ "Abschließen" gesperrt bis alle Items gemappt
→ Vorschau: welche Produkte/Mengen kommen in den Job
→ Bestätigen → Items landen im Job + Mappings global gespeichert
```

---

## Komponente 1: `MappingModal` (React)

**Datei:** `web/src/components/MappingModal.tsx`

### Props
```typescript
interface MappingModalProps {
  uploadId: number;
  jobId?: number;          // null wenn Job noch nicht gespeichert
  onComplete: (items: MappedItem[]) => void;
  onClose: () => void;
}
```

### Verhalten
- Öffnet sich automatisch nach erfolgreichem PDF-Upload in `JobsPage.tsx`
- Ruft `GET /api/pdf/extraction/:upload_id` ab und zeigt alle Items
- Triggert `POST /api/pdf/auto-map/:extraction_id` beim Öffnen (falls noch nicht geschehen)
- **Fortschrittsbalken** oben: "5/8 gemappt"

### Item-Zustände
| Status | Darstellung |
|--------|-------------|
| `auto_mapped` (≥80%) | Grüne Zeile, Produktname + Konfidenz-% Badge |
| `auto_mapped` (60-79%) | Gelbe Zeile, Konfidenz-% Badge, Inline-Suche vorbefüllt |
| `pending` / nicht erkannt | Orange Rand, Inline-Suche leer |
| `user_confirmed` | Grüne Zeile mit ✓-Icon |

### Inline-Suche (Dropdown)
- Debounce 300ms auf `GET /api/pdf/products/search?q=&limit=6`
- Zeigt Produktname + Kategorie-Icon
- Bei Klick: `PUT /api/pdf/items/:item_id/mapping` → `{product_id, mapping_type: "manual"}`
- Mapping wird sofort global gespeichert (`pdf_product_mappings` via Handler)
- Zeile wechselt zu grün/bestätigt

### Footer
- **Gesperrt** (grauer Button) solange `pending`-Items > 0 mit Hinweis "X Items noch nicht gemappt"
- **Aktiv** wenn alle gemappt → Button "Abschließen →" klickbar

### Schritt 2: Vorschau
- Ersetzt Mapping-Liste durch Zusammenfassung:
  - Liste: Produktname · Menge · Stückpreis · Gesamt
  - Gesamtsumme aus PDF
- Buttons: „Zurück" | „Zum Job hinzufügen ✓"
- Bei Bestätigung: `POST /api/pdf/extractions/:extraction_id/finalize` (mit `job_id`)
- `onComplete(items)` wird aufgerufen → JobsPage übernimmt Items in den Job

---

## Komponente 2: Integration in `JobsPage.tsx`

### Änderungen
- Nach erfolgreichem Upload (`upload_id` vorhanden): `MappingModal` öffnen statt bisheriger direkter Verarbeitung
- `onComplete`: gemappte Items werden als Job-Positionen in den Formular-State übernommen (Produkt + Menge)
- Job kann erst gespeichert werden wenn Modal abgeschlossen oder explizit geschlossen (ohne Mapping)
- Beim Schließen ohne Abschluss: Warnung "Mapping nicht abgeschlossen — Items werden nicht hinzugefügt"

---

## Komponente 3: Mapping-Management-Seite

**Route:** `GET /settings/mappings` (bleibt server-rendered, Template wird überarbeitet)  
**Template:** `web/templates/mapping_management.html`

### Layout
- **3 Tabs:** Produkte (n) · Pakete (n) · Kunden (n) — je Tab eigene Tabelle
- **Suchfeld** (Echtzeit-Filter auf OCR-Text und Ziel-Name)
- **Filter-Button:** "Unsichere zuerst" (sortiert nach `confidence_score ASC`)
- **"+ Neu"-Button** pro Tab

### Tabellenstruktur (Produkte-Tab)
| Spalte | Inhalt |
|--------|--------|
| OCR-Text | Raw-Text wie er aus der PDF kam |
| Produkt | Name des gemappten Produkts |
| Konfidenz | Visueller Balken + %-Zahl |
| Typ | Badge: `exakt` / `manuell` / `fuzzy` |
| Nutzung | Zähler wie oft verwendet |
| Aktionen | ✏ Bearbeiten · 🗑 Löschen |

### Bearbeiten (Inline)
- Klick auf ✏: OCR-Text-Zelle wird zur Suche (bestehende Inline-Edit-Logik erweitern)
- Dropdown-Suche wie im MappingModal
- Speichern via `PUT /api/pdf/mappings/:id`

### Neu anlegen
- Kleines Formular oben (erscheint bei "+ Neu"):
  - Freitext-Feld "OCR-Text" + Produktsuche
  - `POST /api/pdf/mappings` (neuer Endpunkt)

### Pakete-Tab und Kunden-Tab
- Identische Struktur, andere Endpunkte:
  - Pakete: `pdf_package_mappings` → `PUT/DELETE /api/pdf/package-mappings/:id`
  - Kunden: `pdf_customer_mappings` → `PUT/DELETE /api/pdf/customer-mappings/:id`

---

## API-Änderungen

### Neu benötigt
| Methode | Route | Zweck |
|---------|-------|-------|
| `GET` | `/api/pdf/extractions/:id/preview` | Liefert Vorschau-Items (Produkt, Menge, Preis) vor Finalisierung |
| `POST` | `/api/pdf/mappings` | Neues Produkt-Mapping ohne Extraktionskontext anlegen |
| `PUT` | `/api/pdf/package-mappings/:id` | Paket-Mapping bearbeiten |
| `DELETE` | `/api/pdf/package-mappings/:id` | Paket-Mapping löschen |
| `PUT` | `/api/pdf/customer-mappings/:id` | Kunden-Mapping bearbeiten |
| `DELETE` | `/api/pdf/customer-mappings/:id` | Kunden-Mapping löschen |

### Bestehend (unverändert nutzbar)
- `POST /api/pdf/upload` ✓
- `GET /api/pdf/extraction/:upload_id` ✓
- `POST /api/pdf/auto-map/:extraction_id` ✓
- `PUT /api/pdf/items/:item_id/mapping` ✓
- `GET /api/pdf/products/search` ✓
- `POST /api/pdf/extractions/:id/finalize` ✓
- `GET /api/pdf/mappings` ✓ (erweitern um Tab-Filter `?type=product|package|customer`)
- `PUT /api/pdf/mappings/:id` ✓
- `DELETE /api/pdf/mappings/:id` ✓

---

## Globales Mapping-Speichern

Jedes manuell gesetzte Mapping im Modal wird sofort via `PUT /api/pdf/items/:item_id/mapping` gespeichert. Der Handler schreibt dabei automatisch in `pdf_product_mappings` (UPSERT), sodass beim nächsten Upload derselbe OCR-Text direkt auto-gemappt wird (`mapping_type = 'manual'`, `confidence = 100`).

Beim Finalisieren (`FinalizeExtraction`) werden alle gemappten Items nochmal als `usage_count++` in die Mapping-Tabelle geschrieben.

---

## Was existiert bereits (nicht neu bauen)

- `PDFHandler` mit allen Core-Funktionen (3486 Zeilen) ✓
- `pdf_product_mappings`, `pdf_package_mappings`, `pdf_customer_mappings` Tabellen ✓
- Auto-Mapping-Algorithmus (Fuzzy + Exact) ✓
- `mapping_management.html` Template (wird überarbeitet, nicht neu) ✓
- `GetAllMappingsAPI`, `DeleteMappingAPI`, `UpdateMappingAPI` ✓
- `SearchProducts`, `SearchPackages` Endpoints ✓

## Was neu gebaut wird

1. **`MappingModal.tsx`** — React-Komponente (Herzstück, neu)
2. **Integration in `JobsPage.tsx`** — Modal einbinden, onComplete-Handler
3. **`GET /api/pdf/extractions/:id/preview`** — neuer Endpunkt
4. **`POST /api/pdf/mappings`** — neuer Endpunkt für manuelles Anlegen
5. **Package/Customer CRUD-Endpoints** — 4 neue Routen
6. **`mapping_management.html` überarbeiten** — Tabs + Konfidenz-Balken-Design

---

## Nicht in diesem Scope

- Kunden-Mapping im Modal (nur Produkte/Pakete werden als Job-Items hinzugefügt)
- Batch-Import mehrerer PDFs
- ML-Training-Loop / Feedback-System
- Pakete-Suche im Modal (nur Produkte)
