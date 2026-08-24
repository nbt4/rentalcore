# RentalCore

**Kernservice für Vermietungsmanagement im Cores-Ökosystem — Auftragsverwaltung, Kundendaten, Gerätezuweisung, Barcode-Generierung und automatisierte OCR-Belegverarbeitung.**

---

## Features

- **Auftragsmanagement (Jobs)** — Vollständiger CRUD-Workflow für Mietaufträge mit Status-Tracking, Gerätezuweisungen und Terminverwaltung
- **Kundenverwaltung** — CRM mit Kontaktdaten, Historie und verknüpften Aufträgen
- **Gerätezuweisung** — Zuweisung und Entfernung von Devices zu/von Aufträgen. Verfügbarkeitsprüfung in Echtzeit
- **Barcode- und QR-Generierung** — Automatische Erstellung von QR-Codes und Barcode-Labels (Barcode128) pro Gerät/Seriennummer
- **OCR-Belegverarbeitung** — Python-3.12-Pipeline in einer geprüften venv zur Extraktion von Positionsdaten; neue Produkt-Katalogentwürfe können sicher und duplikatgeprüft angelegt werden, während Klassifizierung und physische Geräte bewusst in WarehouseCore gepflegt werden
- **M365-Kontaktsync** — Bidirektionale Synchronisation mit Microsoft 365 Shared-Mailbox-Kontakten über die zentrale Cores-App-Registrierung
- **Nextcloud Filepool** — WebDAV-basierte Dateiablage für auftragsbezogene Dokumente mit automatischer Zuweisung
- **Passkey / WebAuthn** — Passwortlose Authentifizierung mit FIDO2/WebAuthn (Passkeys)
- **Analytics Dashboard** — Interaktiver Netto-/Brutto-Umsatz-Drilldown aus den live synchronisierten Auftragspositionen über eigene Produkte, Mietprodukte samt Lieferantenkosten/Marge und Dienstleistungen bis zum konkreten Einzelgerät; Mietkosten folgen dabei der Auftragseinstellung „Preis × Veranstaltungstage“
- **Installierbare Mobile-App (PWA)** — Standalone-Modus mit RentalCore-App-Icon, Safe-Area-Unterstützung, großen Touch-Zielen, App-Tabbar und Drawer-Navigation; auf iPhone/iPad über Safari → Teilen → „Zum Home-Bildschirm“ installieren
- **Zentrales Branding** — Semantische RentalCore-Logos in Sidebar, Login, Favicon und PWA; Rechnungen, HTML-E-Mails und Geräteetiketten verwenden getrennt davon die zentrale Unternehmensmarke

---

## Tech-Stack

| Schicht       | Technologie                                        |
|---------------|----------------------------------------------------|
| Backend       | Go 1.24, Gin, GORM, PostgreSQL 16                  |
| Frontend      | React 19, TypeScript, Vite 7, Zustand 5            |
| Styling       | Tailwind CSS 4, PostCSS, Tailwind Merge            |
| Auth          | JWT (golang-jwt/jwt/v5), bcrypt, WebAuthn          |
| Barcodes      | boombuler/barcode, skip2/go-qrcode                 |
| PDF           | jung-kurt/gofpdf                                   |
| OCR           | Python 3 venv mit custom OCR-Pipeline              |
| Dateiablage   | Nextcloud WebDAV                                   |
| Container     | Docker (Multi-Stage: Node 20 + Go 1.25 + Python 3.12 Alpine) |

---

## Schnellstart

### Docker

```bash
docker run -d \
  --name rentalcore \
  -e DB_HOST=postgres \
  -e DB_PORT=3306 \
  -e DB_NAME=rentalcore \
  -e DB_USERNAME=rentalcore_user \
  -e DB_PASSWORD=yourpassword \
  -e ENCRYPTION_KEY=your-256-bit-encryption-key \
  -e GIN_MODE=release \
  -p 8080:8080 \
  nobentie/rentalcore:latest
```

### docker-compose (Auszug)

```yaml
rentalcore:
  image: nobentie/rentalcore:latest
  ports:
    - "8081:8080"
  environment:
    DB_HOST: postgres
    DB_PORT: 5432
    DB_NAME: rentalcore
    DB_USER: rentalcore
    DB_PASS: ${DB_PASS}
    ENCRYPTION_KEY: ${ENCRYPTION_KEY}
    GIN_MODE: release
    CORES_JWT_SECRET: ${CORES_JWT_SECRET}
    NEXTCLOUD_WEBDAV_URL: ${NEXTCLOUD_WEBDAV_URL}
  depends_on:
    - postgres
  volumes:
    - rental_uploads:/app/uploads
```

---

## API-Endpunkte

| Methode  | Pfad                                        | Beschreibung                                  |
|----------|---------------------------------------------|-----------------------------------------------|
| `POST`   | `/api/v1/auth/login`                        | Benutzer-Login                                |
| `POST`   | `/api/v1/auth/logout`                       | Session beenden                               |
| `GET`    | `/api/v1/auth/me`                           | Aktuellen Benutzer abrufen (🔒)                |
| `POST`   | `/api/v1/auth/change-password`              | Passwort ändern (🔒)                           |
| `GET`    | `/jobs`                                     | Auftragsliste (🔒)                             |
| `POST`   | `/jobs`                                     | Neuen Auftrag erstellen (🔒)                   |
| `GET`    | `/jobs/:id`                                 | Auftragsdetails (🔒)                           |
| `PUT`    | `/jobs/:id`                                 | Auftrag aktualisieren (🔒)                     |
| `DELETE` | `/jobs/:id`                                 | Auftrag löschen (🔒)                           |
| `GET`    | `/jobs/:id/devices`                         | Zugewiesene Geräte abrufen (🔒)                |
| `POST`   | `/jobs/:id/devices`                         | Gerät zuweisen (🔒)                            |
| `DELETE` | `/jobs/:id/devices/:deviceId`               | Gerät entfernen (🔒)                           |
| `GET`    | `/customers`                                | Kundenliste (🔒)                               |
| `POST`   | `/customers`                                | Neuen Kunden anlegen (🔒)                      |
| `PUT`    | `/customers/:id`                            | Kunden aktualisieren (🔒)                      |
| `DELETE` | `/customers/:id`                            | Kunden löschen (🔒)                            |
| `GET`    | `/devices/:id`                              | Gerätedetails (🔒)                             |
| `GET`    | `/devices/:id/stats`                        | Gerätestatistiken (🔒)                         |
| `GET`    | `/devices/available`                        | Verfügbare Geräte (🔒)                         |
| `GET`    | `/barcodes/device/:serialNo/qr`             | QR-Code pro Gerät generieren (🔒)              |
| `GET`    | `/barcodes/device/:serialNo/barcode`        | Barcode pro Gerät generieren (🔒)              |
| `GET`    | `/analytics/revenue`                        | Umsatz-Analytics (🔒)                          |
| `GET`    | `/analytics/revenue/drilldown`              | Hierarchischer Umsatz-Drilldown (🔒)           |
| `GET`    | `/analytics/equipment`                      | Equipment-Analytics (🔒)                       |
| `GET`    | `/statuses`                                 | Status-Liste (🔒)                              |
| `GET`    | `/health`                                   | Health Check (öffentlich)                      |

🔒 = Authentifizierung via `session_id` Cookie erforderlich

---

## Umgebungsvariablen

| Variable                       | Beschreibung                                      | Standard               |
|--------------------------------|---------------------------------------------------|------------------------|
| `DB_HOST`                      | Datenbank-Host                                    | –                      |
| `DB_PORT`                      | Datenbank-Port                                    | `3306`                 |
| `DB_NAME`                      | Datenbank-Name                                    | `rentalcore`           |
| `DB_USERNAME`                  | Datenbank-Benutzer                                | –                      |
| `DB_PASSWORD`                  | Datenbank-Passwort                                | –                      |
| `ENCRYPTION_KEY`               | 256-Bit-Verschlüsselungs-Key                      | –                      |
| `SESSION_TIMEOUT`              | Session-Timeout in Sekunden                       | `3600`                 |
| `GIN_MODE`                     | Gin-Modus (`release` oder `debug`)                | `release`              |
| `NEXTCLOUD_WEBDAV_URL`         | Nextcloud WebDAV-URL für Filepool                 | –                      |
| `NEXTCLOUD_WEBDAV_USER`        | Nextcloud WebDAV-Benutzer                         | –                      |
| `NEXTCLOUD_WEBDAV_PASSWORD`    | Nextcloud WebDAV-Passwort                         | –                      |
| `NEXTCLOUD_WEBDAV_BASE_PATH`   | WebDAV-Basispfad                                  | `rentalcore-filepool`  |
| `FILEPOOL_ASSIGNED_ROOT`       | Root-Pfad für zugewiesene Dateien                 | `assigned`             |
| `FILEPOOL_UNASSIGNED_ROOT`     | Root-Pfad für unzugewiesene Dateien               | `unassigned`           |
| `SMTP_HOST`                    | SMTP-Host für E-Mail-Versand                      | –                      |
| `SMTP_PORT`                    | SMTP-Port                                         | `587`                  |
| `SMTP_USERNAME`                | SMTP-Benutzername                                 | –                      |
| `SMTP_PASSWORD`                | SMTP-Passwort                                     | –                      |
| `M365_TENANT_ID`               | Entra ID Tenant-ID (für Kontaktsync)              | –                      |
| `M365_CLIENT_ID`               | Entra ID Client-ID                                | –                      |
| `M365_CLIENT_SECRET`           | Entra ID Client-Secret                            | –                      |
| `M365_SHARED_MAILBOX_ID`       | Shared Mailbox-ID                                 | –                      |
| `M365_SYNC_INTERVAL`           | Sync-Intervall (z. B. `5m`)                       | `5m`                   |
| `WAREHOUSECORE_DOMAIN`         | WarehouseCore-Domain für Cross-Navigation         | –                      |
| `CORES_JWT_SECRET`             | JWT-Secret (Cores-weit identisch)                 | –                      |

Die `M365_*`-Variablen bleiben als Fallback bestehen. Sobald im Cores-Dashboard unter **Microsoft 365 & Entra** Werte gespeichert sind, lädt RentalCore Tenant-ID, Client-ID, Secret, Mailboxen, Intervall und App-URL beim Start aus der gemeinsamen Tabelle `m365_settings`. Damit wird für Entra-Benutzer, Login, Kontakte und Kalender nur eine registrierte Tenant-App benötigt.

---

[Quellcode](https://github.com/nbt4/rentalcore) | [Monorepo](https://github.com/nbt4/cores) | `nobentie/rentalcore:latest`
