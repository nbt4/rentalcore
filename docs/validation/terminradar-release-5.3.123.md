# Terminradar: Release-Prüfung für RentalCore 5.3.123

Basis ist das aktuelle RentalCore-main `e969e64`, das den produktiven
Stand 5.3.122 einschließlich Fremdmiet-Zuordnungen enthält. Das Release hebt nur
die Versionskonstante an und ergänzt die getestete Terminradar-Korrektur.
Die Gates liefen in der vorgeschriebenen Reihenfolge. Die vorhandenen 43
Formatabweichungen sind vom Nutzer für diese Arbeit ausdrücklich freigegeben;
Lint ist auf dieser Basis wieder grün. Es wurden keine Tests gelockert oder
neuen Paketabhängigkeiten hinzugefügt.

Die Go-Prüfung lief ohne Cache. Ohne konfigurierte Testdatenbank überspringen
sich die bestehenden PostgreSQL-Integrationstests selbst. Der Terminradar-Fix
ändert kein Backend-Verhalten, keine SQL-Migrationen und keinen Start-Schema-Code.
Im Browser wurden nur erfundene Daten benutzt; keine Produktionstests oder
Fachmutationen fanden statt.

Ein lokaler Docker-Kandidat `rentalcore-terminradar:verify` wurde aus der
aktuellen Basis und den ausdrücklich ausgewählten Fix-Dateien erfolgreich
gebaut. Die unveränderte Dockerfile-Prüfung der OCR-Imports lief erfolgreich.
Seine Image-ID ist `sha256:723d169f271ad21549f1f668971f463a45c48abba824fbd807ae99e6431ea436`.
Dieser Kandidat ist noch kein veröffentlichtes Release-Abbild. Das endgültige
Abbild muss aus dem eingecheckten Dienst-Commit mit korrektem OCI-Revisionslabel
entstehen und darf erst nach dem Merge veröffentlicht werden.

Die suiteweiten Gates bestehen auf dem sauberen Suite-main `c11fe81`.
Insbesondere sind dessen Designsystem-Kopien aktuell. Die frühere Prüfung
gegen den bestehenden Entwicklungs-Workspace mit veränderten PlannerCore-Dateien
ist kein Befund dieses Releases.

Die folgende Ausgabe stammt aus den erneuten tatsächlichen Läufen.

## Format: bestehende 43 Abweichungen, Fortsetzung vom Nutzer freigegeben

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

## Frontend-Lint: grün, fünf bestehende Warnungen

`cd web && npm run lint`

```text

> rentalcore-web@0.0.0 lint
> eslint .


/opt/dev/rentalcore-terminradar/web/src/pages/JobsPage.tsx
  996:32  warning  React Hook useEffect has a missing dependency: 'load'. Either include it or remove the dependency array  react-hooks/exhaustive-deps

/opt/dev/rentalcore-terminradar/web/static/scanner/ui/ScannerView.tsx
  216:8  warning  React Hook useCallback has missing dependencies: 'setupEventListeners' and 'startScanning'. Either include them or remove the dependency array                                                                                                                      react-hooks/exhaustive-deps
  239:8  warning  React Hook useCallback has missing dependencies: 'handleCameraError', 'handleCameraFrame', 'handleDecodeResult', 'handleDecoderError', 'handleDoubleTapGesture', 'handleFocusGesture', and 'handleZoomGesture'. Either include them or remove the dependency array  react-hooks/exhaustive-deps
  264:8  warning  React Hook useCallback has a missing dependency: 'showScanFeedback'. Either include it or remove the dependency array                                                                                                                                               react-hooks/exhaustive-deps
  288:8  warning  React Hook useCallback has a missing dependency: 'calculateROI'. Either include it or remove the dependency array                                                                                                                                                   react-hooks/exhaustive-deps

✖ 5 problems (0 errors, 5 warnings)
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
✓ 1767 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-TVeDGtA7.js   489.70 kB │ gzip: 145.56 kB
✓ built in 6.48s
```

## Go-Build: grün

`make build`

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

## Go-Tests ohne Cache: grün

`go test ./... -count=1`

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.025s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.005s
ok  	go-barcode-webapp/internal/handlers	0.047s
ok  	go-barcode-webapp/internal/jev	0.061s
ok  	go-barcode-webapp/internal/jobstatus	0.059s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.051s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.055s
ok  	go-barcode-webapp/internal/services	0.078s
ok  	go-barcode-webapp/internal/services/pdf	0.037s
ok  	go-barcode-webapp/internal/services/postalcode	0.019s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.022s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Vet: grün

`go vet ./...`

```text
(leere Ausgabe)
```

## Regressionstests mit Node 20.19.5: 9/9 grün

`node --test web/tests/dashboard.test.mjs`

```text
TAP version 13
# (node:336890) ExperimentalWarning: The MockTimers API is an experimental feature and might change at any time
# (Use `node --trace-warnings ...` to show where the warning was created)
# Subtest: overdue jobs cannot displace current and upcoming appointments
ok 1 - overdue jobs cannot displace current and upcoming appointments
  ---
  duration_ms: 6.983447
  ...
# Subtest: completed, cancelled and undated jobs stay out of the radar
ok 2 - completed, cancelled and undated jobs stay out of the radar
  ---
  duration_ms: 0.345753
  ...
# Subtest: running jobs precede the next five appointments without changing the source list
ok 3 - running jobs precede the next five appointments without changing the source list
  ---
  duration_ms: 0.357466
  ...
# Subtest: a job ending today leaves the radar after the local day changes
ok 4 - a job ending today leaves the radar after the local day changes
  ---
  duration_ms: 0.379245
  ...
# Subtest: end-only dates are ordered with upcoming jobs
ok 5 - end-only dates are ordered with upcoming jobs
  ---
  duration_ms: 0.298162
  ...
# Subtest: day boundaries use the local calendar date
ok 6 - day boundaries use the local calendar date
  ---
  duration_ms: 0.323595
  ...
# Subtest: visible dashboards refresh every minute and on focus
ok 7 - visible dashboards refresh every minute and on focus
  ---
  duration_ms: 3.155592
  ...
# Subtest: hidden tabs pause refresh and reload when visible again
ok 8 - hidden tabs pause refresh and reload when visible again
  ---
  duration_ms: 1.152469
  ...
# Subtest: leaving the dashboard removes timers and browser listeners
ok 9 - leaving the dashboard removes timers and browser listeners
  ---
  duration_ms: 0.913348
  ...
1..9
# tests 9
# suites 0
# pass 9
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 1768.381593
```

## Lokaler Chromium: grün

`SPA-Build mit ausschließlich erfundenen API-Antworten`

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
