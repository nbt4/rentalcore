# Personalzuordnung: Prüfprotokoll

2026-10-16. Ausschließlich synthetische Daten im lokalen PostgreSQL16-Compose-
Stack `cores-staffing-lifecycle-test`, Datenbank `cores_staffing_lifecycle_test`.
Root-Migrationen bis053 plus die bestehenden Warehouse-Runtime-Migrationen039,
040,044 und kanonische Product-Master-Initialisierung vervollständigen die
isolierte gemeinsame Datenbank. Jeder Owner-Test baut sein eigenes Schema auf
und installiert Startup054 zweimal. Zusätzlich frischer separater Compose-Stack
`cores-staffing-clean-test` mit allen Root-Migrationen bis054.
Keine produktiven Fachmutationen oder privaten Testdaten.

## Vollständige Tests

`go test -count=1 ./...`

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.026s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.010s
ok  	go-barcode-webapp/internal/handlers	61.822s
ok  	go-barcode-webapp/internal/jev	0.040s
ok  	go-barcode-webapp/internal/jobstatus	0.017s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.220s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	2.669s
ok  	go-barcode-webapp/internal/services	0.042s
ok  	go-barcode-webapp/internal/services/pdf	0.041s
ok  	go-barcode-webapp/internal/services/postalcode	0.017s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.046s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Race

`go test -race -count=1 ./internal/handlers ./internal/schema ./internal/repository ./internal/sync/m365`

```text
ok  	go-barcode-webapp/internal/handlers	156.133s
ok  	go-barcode-webapp/internal/schema	3.030s
ok  	go-barcode-webapp/internal/repository	1.510s
ok  	go-barcode-webapp/internal/sync/m365	1.143s
```

`go vet ./...` und `go build -o /tmp/staff-rentalcore-server ./cmd/server`
beenden sich erfolgreich ohne Ausgabe.

Die neuen Regressionen prüfen atomare native/MCP-Audits und Jobhistorie,
Idempotenz/Rechteentzug vor Replay, genaue Versionen und geänderten Mitarbeiter-/
Konfliktkontext, Rollen/Clear/Unicodegrenzen, inaktive Mitarbeiter, fremde aktive
Bearbeiter, Überschneidungsbestätigung, retained Archive/Restore und DB-Guards.
Echte Zwei-Verbindungs-Tests prüfen Mitarbeitersperren in beiden Reihenfolgen,
NOWAIT-Tabellenkollision und gleichzeitigen Create-Replay mit nur einer Identität.
Der Schema-Test prüft identische Migration und begrenzt wartendes Startup-DDL.
Kalender-Tests prüfen Callback erst nach Auditcommit, keinen Vorschau-/Replay-
Callback und spätere Bereinigung archivierter Legacy-Termine.

## Frische Root-Migration054

Nach Abschluss der PostgreSQL-Initialisierung:

```text
 column_name |          data_type
-------------+-----------------------------
 deleted_at  | timestamp without time zone
 updated_at  | timestamp without time zone
(2 rows)

                 tgname
----------------------------------------
 job_employees_guard_staffing_lifecycle
(1 row)

```

## Frontend

`docker run --rm -v /opt/dev/rentalcore-staffing-lifecycle/web:/app -w /app node:22-alpine sh -c 'npm ci && npm run lint && npm run build'`

Lint:0 Fehler,5 bereits bestehende Hook-Warnungen. Build erfolgreich:

```text
vite v7.3.1 building client environment for production...
transforming...
Browserslist: browsers data (caniuse-lite) is 7 months old. Please run:
  npx update-browserslist-db@latest
  Why you should do it regularly: https://github.com/browserslist/update-db#readme
✓ 1769 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-BCWpXjY8.js   490.18 kB │ gzip: 145.75 kB
✓ built in 11.21s
```

`npm ci` meldet16 bestehende Auditbefunde (2low,3moderate,11high) bei unverändertem
Lockfile. Keine Abhängigkeits- oder UI-Quelländerung in diesem Auftrag;
der generierte Index wurde nach dem Nachweis auf den Commitstand zurückgesetzt.

Unabhängiges Review und Nachreview ohne verbleibende konkrete Findings.
Vier Befunde (Kalender-Retry, Mitarbeiter-Deaktivierungsrennen, Root-Migrationspfad,
Startup-DDL-Zeitlimits) wurden vor Freigabe behoben und gezielt geprüft.
