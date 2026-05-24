# Design: Veranstaltungsorte (Venues)

**Datum:** 2026-05-24  
**Status:** Approved

---

## Überblick

Einführung von Veranstaltungsorten (Venues) als eigene Entität. Venues können im Dashboard verwaltet und optional einem Job zugewiesen werden. Ist ein Venue gesetzt, wird dessen Adresse priorisiert als Ort im M365-Kalendereintrag verwendet; andernfalls greift die Kundenadresse.

---

## 1. Datenmodell

### Neue Tabelle: `venues`

| Spalte          | Typ           | Pflicht | Beschreibung                    |
|-----------------|---------------|---------|----------------------------------|
| `id`            | SERIAL PK     | ja      | Primärschlüssel                  |
| `name`          | VARCHAR(255)  | ja      | Name des Veranstaltungsorts      |
| `street`        | VARCHAR(255)  | nein    | Straße                           |
| `house_number`  | VARCHAR(50)   | nein    | Hausnummer                       |
| `zip`           | VARCHAR(20)   | nein    | PLZ                              |
| `city`          | VARCHAR(255)  | nein    | Stadt                            |
| `contact_name`  | VARCHAR(255)  | nein    | Ansprechpartner (Name)           |
| `phone`         | VARCHAR(100)  | nein    | Telefonnummer                    |
| `email`         | VARCHAR(255)  | nein    | E-Mail-Adresse                   |
| `notes`         | TEXT          | nein    | Interne Notizen                  |
| `created_at`    | TIMESTAMP     | ja      | Auto-gesetzt                     |
| `updated_at`    | TIMESTAMP     | ja      | Auto-gesetzt                     |

### Änderung: Tabelle `jobs`

Neues nullable Feld:

| Spalte      | Typ        | Beschreibung                              |
|-------------|------------|-------------------------------------------|
| `venue_id`  | INTEGER FK | Referenz auf `venues.id`, nullable        |

GORM-Tag: `gorm:"column:venue_id"` mit entsprechendem `Venue *Venue` Relation-Feld.

### Migration

Neue Datei: `migrations/postgresql/003_venues.sql`

```sql
CREATE TABLE venues (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    street      VARCHAR(255),
    house_number VARCHAR(50),
    zip         VARCHAR(20),
    city        VARCHAR(255),
    contact_name VARCHAR(255),
    phone       VARCHAR(100),
    email       VARCHAR(255),
    notes       TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE jobs ADD COLUMN venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;
```

---

## 2. Backend

### Go-Modell

```go
type Venue struct {
    ID          uint    `json:"id" gorm:"primaryKey"`
    Name        string  `json:"name" gorm:"column:name;not null"`
    Street      *string `json:"street" gorm:"column:street"`
    HouseNumber *string `json:"house_number" gorm:"column:house_number"`
    ZIP         *string `json:"zip" gorm:"column:zip"`
    City        *string `json:"city" gorm:"column:city"`
    ContactName *string `json:"contact_name" gorm:"column:contact_name"`
    Phone       *string `json:"phone" gorm:"column:phone"`
    Email       *string `json:"email" gorm:"column:email"`
    Notes       *string `json:"notes" gorm:"column:notes"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Job-Modell erhält:
```go
VenueID *uint  `json:"venue_id" gorm:"column:venue_id"`
Venue   *Venue `json:"venue,omitempty" gorm:"foreignKey:VenueID"`
```

### Repository: `venue_repository.go`

- `List() ([]Venue, error)`
- `GetByID(id uint) (*Venue, error)`
- `Create(v *Venue) error`
- `Update(v *Venue) error`
- `Delete(id uint) error`

### Handler: `venue_handler.go`

REST-Endpunkte unter `/api/v1/venues`:

| Methode | Pfad                 | Funktion          |
|---------|----------------------|-------------------|
| GET     | `/api/v1/venues`     | Liste aller Venues |
| POST    | `/api/v1/venues`     | Venue anlegen     |
| PUT     | `/api/v1/venues/:id` | Venue bearbeiten  |
| DELETE  | `/api/v1/venues/:id` | Venue löschen     |

Page-Route: `GET /venues` → rendert `venues.html`

---

## 3. Kalender-Integration

In `calendar_sync.go`, Methode `buildEvent`:

**Priorität für `location`-Feld im M365-Event:**

1. Job hat `VenueID` gesetzt → Venue-Adresse formatieren: `"Name, Straße Hausnr, PLZ Stadt"`
2. Kein Venue → Kundenadresse prüfen: `"Straße Hausnr, PLZ Stadt"` (nur wenn mindestens ein Adressfeld gesetzt)
3. Kein Venue, keine Kundenadresse → `location` wird nicht gesetzt

Das `CalendarEvent`-Struct erhält ein neues Feld:
```go
Location *EventLocation `json:"location,omitempty"`

type EventLocation struct {
    DisplayName string `json:"displayName"`
}
```

---

## 4. Frontend

### Sidebar-Navigation (`base.html`)

Neuer Eintrag zwischen Customers und Invoices:
```html
<a href="/venues" class="rc-nav-item {{if eq .currentPage "venues"}}active{{end}}">
    <i class="bi bi-geo-alt"></i>
    <span class="rc-nav-label">Veranstaltungsorte</span>
</a>
```

### Template: `venues.html`

- Listenansicht analog `customers.html`: Tabelle mit Name, Stadt, Ansprechpartner
- „Neu anlegen"-Button öffnet Modal mit Formular (alle Felder)
- Inline-Bearbeiten und Löschen per Aktionsbuttons
- AJAX-basiert (fetch API), kein Seiten-Reload

### Job-Formular (`job_form.html` / Job-Detail)

- Combobox-Feld „Veranstaltungsort" (optional):
  - Zeigt alle Venues wenn leer/Klick
  - Filtert beim Tippen
  - „Kein Veranstaltungsort" als Default-Option (leerer Wert)
- Kleiner Hinweistext unterhalb: welche Adresse im Kalender erscheint
  - Venue gewählt → „Kalender-Ort: {Venue-Name}, {Stadt}"
  - Kein Venue + Kundenadresse → „Kalender-Ort: Kundenadresse ({Stadt})"
  - Kein Venue, keine Kundenadresse → „Kein Ort im Kalender"

---

## 5. Fehlerbehandlung

- Venue löschen: Prüfen ob noch Jobs referenzieren → Warnung im Frontend, Löschen trotzdem erlaubt (FK `ON DELETE SET NULL`)
- Venue-Name Pflichtfeld: Frontend-Validierung + Backend-Validierung
- Kalender: fehlendes `location`-Feld ist kein Fehler, Event wird ohne Ort erstellt

---

## 6. Scope

**In Scope:**
- Venue CRUD (Backend + Frontend)
- `venue_id` auf Jobs
- Venue-Combobox im Job-Formular
- Kalender-Location aus Venue oder Kundenadresse
- Migration SQL-Datei

**Out of Scope:**
- Karte/Geo-Visualisierung
- Venue-spezifische Kalender-Kategorien
- Mehrere Adressen pro Venue
