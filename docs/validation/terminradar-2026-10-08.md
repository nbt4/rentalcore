# Terminradar: Testprotokoll vom 8. Oktober 2026

Getestet im isolierten RentalCore-Worktree auf Basis von `dcc87a8`.
Die Pflichtstufen liefen nacheinander. Der Nutzer hat die Fortsetzung trotz
bestehender Formatfehler und fehlender ESLint-Konfiguration ausdrücklich
freigegeben. Die ursprünglichen Dateien wurden dafür nicht verändert.

Frontend-Build, Go-Build, Go-Tests, Vet und neun neue Regressionstests sind grün.
Die Go-Tests liefen ohne Cache mit `-count=1`. Eine Testdatenbank ist in dieser
Umgebung nicht konfiguriert; die bereits vorhandenen PostgreSQL-Integrationstests
überspringen sich deshalb selbst. Für die Dashboard-Korrektur wurden keine
Datenbank oder echten Fremdschnittstellen benutzt.

Die suiteweite Designsystemprüfung ist im vorhandenen Umbrella-Workspace wegen
veralteter PlannerCore-Kopien rot. Der gesonderte Vergleich aller sechs
RentalCore-Kopien mit den kanonischen Quellen ist grün. Es wurde kein PR erstellt.

Im lokalen Chromium-Browser wurden ausschließlich erfundene API-Antworten
verwendet. Die unveränderte Version reproduziert beide Fehler. Die korrigierte
Version besteht die Prüfungen für aktuelle Termine, Fokus, Timer, Sichtbarkeit,
parallele Anfragen, Fehler/Retry, Tageswechsel und Aufräumen beim Navigieren.
Screenshots wurden bei 390, 768, 1280 und 1536 px in Dark und Light aufgenommen;
Radarlinks wurden per Tastatur geprüft. Die vorhandene Shell und deren Layout-
und Light-Kontrastabweichungen sind durch diesen Verhaltensfix nicht verändert.
Die Browser-Fixture bedient nur den SPA-Build, keine Logo- oder Schriftassets.

Die folgenden Ausgaben stammen aus tatsächlichen lokalen Läufen.

## Format: 43 vorhandene Abweichungen

`gofmt -l .`

```text
cmd/compliance/main.go
internal/compliance/audit_logger.go
internal/compliance/digital_signature.go
internal/compliance/gdpr_compliance.go
internal/compliance/gobd_compliance.go
internal/compliance/middleware.go
internal/compliance/retention_manager.go
internal/handlers/auth_handler.go
internal/handlers/barcode_handler.go
internal/handlers/case_handler.go
internal/handlers/device_handler.go
internal/handlers/document_handler.go
internal/handlers/equipment_package_handler.go
internal/handlers/error_handler.go
internal/handlers/financial_handler.go
internal/handlers/invoice_handler.go
internal/handlers/invoice_template_handler.go
internal/handlers/job_history_handler.go
internal/handlers/monitoring_handler.go
internal/handlers/pwa_handler.go
internal/handlers/scanboard_handler.go
internal/handlers/security_handler.go
internal/handlers/status_handler.go
internal/handlers/types.go
internal/handlers/webauthn_handler.go
internal/logger/structured_logger.go
internal/middleware/performance.go
internal/models/accessories_consumables.go
internal/models/employee.go
internal/models/invoice_models.go
internal/models/job_package.go
internal/monitoring/error_tracker.go
internal/repository/case_repository.go
internal/repository/customer_repository.go
internal/repository/database.go
internal/repository/equipment_package_repository.go
internal/repository/job_attachment_repository.go
internal/repository/job_category_repository.go
internal/routes/scan_fallback.go
internal/scan/decode.go
internal/services/barcode_service.go
internal/services/pdf_service.go
tools/genhash.go
```

## Frontend-Lint: vorhandene Konfiguration fehlt

`cd web && npm run lint`

```text

> rentalcore-web@0.0.0 lint
> eslint .


Oops! Something went wrong! :(

ESLint: 9.39.4

ESLint couldn't find an eslint.config.(js|mjs|cjs) file.

From ESLint v9.0.0, the default configuration file is now eslint.config.js.
If you are using a .eslintrc.* file, please follow the migration guide
to update your configuration file to the new format:

https://eslint.org/docs/latest/use/configure/migration-guide

If you still have problems after following the migration guide, please stop by
https://eslint.org/chat/help to chat with the team.
```

## Frontend-Build: grün

`cd web && npm run build`

```text

> rentalcore-web@0.0.0 build
> tsc -b && vite build

vite v7.3.1 building client environment for production...
transforming...
Browserslist: browsers data (caniuse-lite) is 7 months old. Please run:
  npx update-browserslist-db@latest
  Why you should do it regularly: https://github.com/browserslist/update-db#readme
✓ 1766 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-C7cM0eSa.js   489.66 kB │ gzip: 145.54 kB
✓ built in 6.41s
```

## Go-Build: grün

`make build`

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

## Go-Tests: grün, ohne Cache

`go test ./... -count=1`

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.035s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.008s
ok  	go-barcode-webapp/internal/handlers	0.053s
ok  	go-barcode-webapp/internal/jev	0.029s
ok  	go-barcode-webapp/internal/jobstatus	0.014s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.041s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.044s
ok  	go-barcode-webapp/internal/services	0.026s
ok  	go-barcode-webapp/internal/services/pdf	0.032s
ok  	go-barcode-webapp/internal/services/postalcode	0.015s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.029s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Vet: grün

`go vet ./...`

```text
(leere Ausgabe)
```

## Neue Regressionstests: 9/9 grün mit Node 20.19.5

`node --test web/tests/dashboard.test.mjs`

```text
TAP version 13
# (node:321457) ExperimentalWarning: The MockTimers API is an experimental feature and might change at any time
# (Use `node --trace-warnings ...` to show where the warning was created)
# Subtest: overdue jobs cannot displace current and upcoming appointments
ok 1 - overdue jobs cannot displace current and upcoming appointments
  ---
  duration_ms: 6.455319
  ...
# Subtest: completed, cancelled and undated jobs stay out of the radar
ok 2 - completed, cancelled and undated jobs stay out of the radar
  ---
  duration_ms: 0.303196
  ...
# Subtest: running jobs precede the next five appointments without changing the source list
ok 3 - running jobs precede the next five appointments without changing the source list
  ---
  duration_ms: 0.320662
  ...
# Subtest: a job ending today leaves the radar after the local day changes
ok 4 - a job ending today leaves the radar after the local day changes
  ---
  duration_ms: 0.336304
  ...
# Subtest: end-only dates are ordered with upcoming jobs
ok 5 - end-only dates are ordered with upcoming jobs
  ---
  duration_ms: 0.285789
  ...
# Subtest: day boundaries use the local calendar date
ok 6 - day boundaries use the local calendar date
  ---
  duration_ms: 0.330408
  ...
# Subtest: visible dashboards refresh every minute and on focus
ok 7 - visible dashboards refresh every minute and on focus
  ---
  duration_ms: 3.132199
  ...
# Subtest: hidden tabs pause refresh and reload when visible again
ok 8 - hidden tabs pause refresh and reload when visible again
  ---
  duration_ms: 1.098619
  ...
# Subtest: leaving the dashboard removes timers and browser listeners
ok 9 - leaving the dashboard removes timers and browser listeners
  ---
  duration_ms: 1.121784
  ...
1..9
# tests 9
# suites 0
# pass 9
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 1667.530169
```

## Browser: beide Fehler ohne Fix reproduziert

`Lokaler Chromium gegen SPA-Build aus unverändertem HEAD`

```text
EXPECTED FAIL: overdue jobs displace current/upcoming appointments in unchanged HEAD
EXPECTED FAIL: unchanged HEAD does not reload on focus
```

## Browser: korrigierter SPA-Build

`Lokaler Chromium mit ausschließlich erfundenen API-Antworten`

```text
PASS: current/upcoming radar; overdue jobs remain in work queue
PASS: radar visible at 390, 768, 1280 and 1536 px
PASS: light/dark screenshots and keyboard access to radar links
PASS: focus reloads changed job data
PASS: periodic reload updates visible job data
PASS: hidden tabs pause polling; visible tabs reload immediately
PASS: concurrent refreshes deduplicated; existing data stays visible
PASS: refresh errors retain data and retry recovers
PASS: day rollover recalculates schedule even with unchanged jobs
PASS: leaving dashboard stops polling; no browser runtime errors
```

## Suite-Designsystem: vorhandene PlannerCore-Abweichungen

`/opt/dev/cores/scripts/check-design-system.sh`

```text
Designsystem-Kopie ist nicht aktuell: plannercore/web/src/cores-theme.css
Designsystem-Helfer ist nicht aktuell: plannercore/web/src/lib/cores-design.ts
Sprachumschalter-Kopie ist nicht aktuell: plannercore/web/src/lib/SuiteLanguageSwitcher.tsx
Deutsche Basisübersetzung ist nicht aktuell: plannercore/web/src/lib/cores-locales/de.json
Englische Basisübersetzung ist nicht aktuell: plannercore/web/src/lib/cores-locales/en.json
```

## RentalCore-Designsystem: grün

`Bytevergleich der sechs RentalCore-Kopien mit cores/theme/`

```text
RentalCore: all six design system copies match canonical sources.
```
