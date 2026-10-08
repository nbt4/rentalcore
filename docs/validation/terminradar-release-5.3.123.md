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

Das endgültige Release-Abbild entsteht aus dem finalen eingecheckten Dienst-Commit
mit korrektem OCI-Revisionslabel. Build-Nachweis und Image-ID werden im
Suite-Release-Protokoll festgehalten. Veröffentlichung erfolgt erst nach Merge.

Die suiteweiten Gates bestehen auf dem sauberen Suite-main `c11fe81`.
Insbesondere sind dessen Designsystem-Kopien aktuell. Die frühere Prüfung
gegen den bestehenden Entwicklungs-Workspace mit veränderten PlannerCore-Dateien
ist kein Befund dieses Releases.

Zwölf Regressionstests prüfen zusätzlich den gemeinsamen Abbruch nach einem
Teilfehler und beim Aufräumen. Ohne den Teilfehler-Fix schlägt genau dieser
Regressionstest fehl (11 bestanden, 1 fehlgeschlagen). Die Browserprüfung bestätigt
den sichtbaren Fehlerzustand, Wiederholung und Abbruch beider Anfragen bei Navigation.

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
✓ 1768 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-1XhhrN_R.js   489.80 kB │ gzip: 145.61 kB
✓ built in 6.64s
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
ok  	go-barcode-webapp/cmd/server	0.022s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.006s
ok  	go-barcode-webapp/internal/handlers	0.048s
ok  	go-barcode-webapp/internal/jev	0.021s
ok  	go-barcode-webapp/internal/jobstatus	0.006s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.038s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.044s
ok  	go-barcode-webapp/internal/services	0.028s
ok  	go-barcode-webapp/internal/services/pdf	0.034s
ok  	go-barcode-webapp/internal/services/postalcode	0.019s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.040s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Vet: grün

`go vet ./...`

```text
(leere Ausgabe)
```

## Regressionstests mit Node 20.19.5: 12/12 grün

`node --test web/tests/dashboard.test.mjs`

```text
TAP version 13
# (node:346407) ExperimentalWarning: The MockTimers API is an experimental feature and might change at any time
# (Use `node --trace-warnings ...` to show where the warning was created)
# Subtest: overdue jobs cannot displace current and upcoming appointments
ok 1 - overdue jobs cannot displace current and upcoming appointments
  ---
  duration_ms: 6.170533
  ...
# Subtest: completed, cancelled and undated jobs stay out of the radar
ok 2 - completed, cancelled and undated jobs stay out of the radar
  ---
  duration_ms: 0.307372
  ...
# Subtest: running jobs precede the next five appointments without changing the source list
ok 3 - running jobs precede the next five appointments without changing the source list
  ---
  duration_ms: 0.349771
  ...
# Subtest: a job ending today leaves the radar after the local day changes
ok 4 - a job ending today leaves the radar after the local day changes
  ---
  duration_ms: 0.362144
  ...
# Subtest: end-only dates are ordered with upcoming jobs
ok 5 - end-only dates are ordered with upcoming jobs
  ---
  duration_ms: 0.284494
  ...
# Subtest: day boundaries use the local calendar date
ok 6 - day boundaries use the local calendar date
  ---
  duration_ms: 0.321371
  ...
# Subtest: visible dashboards refresh every minute and on focus
ok 7 - visible dashboards refresh every minute and on focus
  ---
  duration_ms: 3.356648
  ...
# Subtest: hidden tabs pause refresh and reload when visible again
ok 8 - hidden tabs pause refresh and reload when visible again
  ---
  duration_ms: 1.133048
  ...
# Subtest: leaving the dashboard removes timers and browser listeners
ok 9 - leaving the dashboard removes timers and browser listeners
  ---
  duration_ms: 0.962083
  ...
# Subtest: dashboard data returns the fresh jobs and customers together
ok 10 - dashboard data returns the fresh jobs and customers together
  ---
  duration_ms: 0.76751
  ...
# Subtest: a failed request cancels its still-pending partner and preserves the original error
ok 11 - a failed request cancels its still-pending partner and preserves the original error
  ---
  duration_ms: 2.073894
  ...
# Subtest: cleanup cancels both pending data requests
ok 12 - cleanup cancels both pending data requests
  ---
  duration_ms: 0.917561
  ...
1..12
# tests 12
# suites 0
# pass 12
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 1698.81458
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
PASS: failed jobs request aborts its pending customer request
PASS: day rollover recalculates schedule even with unchanged jobs
PASS: leaving dashboard stops polling; no browser runtime errors
PASS: navigation aborts both pending dashboard requests
```

## Negativnachweis: Teilfehlertest ohne Fix

Nur eine isolierte Testkopie wurde verändert; der Release-Quellcode blieb erhalten.

```text
TAP version 13
# (node:346437) ExperimentalWarning: The MockTimers API is an experimental feature and might change at any time
# (Use `node --trace-warnings ...` to show where the warning was created)
# Subtest: overdue jobs cannot displace current and upcoming appointments
ok 1 - overdue jobs cannot displace current and upcoming appointments
  ---
  duration_ms: 6.441866
  ...
# Subtest: completed, cancelled and undated jobs stay out of the radar
ok 2 - completed, cancelled and undated jobs stay out of the radar
  ---
  duration_ms: 0.372588
  ...
# Subtest: running jobs precede the next five appointments without changing the source list
ok 3 - running jobs precede the next five appointments without changing the source list
  ---
  duration_ms: 0.429414
  ...
# Subtest: a job ending today leaves the radar after the local day changes
ok 4 - a job ending today leaves the radar after the local day changes
  ---
  duration_ms: 0.34172
  ...
# Subtest: end-only dates are ordered with upcoming jobs
ok 5 - end-only dates are ordered with upcoming jobs
  ---
  duration_ms: 0.321188
  ...
# Subtest: day boundaries use the local calendar date
ok 6 - day boundaries use the local calendar date
  ---
  duration_ms: 0.39891
  ...
# Subtest: visible dashboards refresh every minute and on focus
ok 7 - visible dashboards refresh every minute and on focus
  ---
  duration_ms: 3.50321
  ...
# Subtest: hidden tabs pause refresh and reload when visible again
ok 8 - hidden tabs pause refresh and reload when visible again
  ---
  duration_ms: 1.168838
  ...
# Subtest: leaving the dashboard removes timers and browser listeners
ok 9 - leaving the dashboard removes timers and browser listeners
  ---
  duration_ms: 1.052968
  ...
# Subtest: dashboard data returns the fresh jobs and customers together
ok 10 - dashboard data returns the fresh jobs and customers together
  ---
  duration_ms: 0.795898
  ...
# Subtest: a failed request cancels its still-pending partner and preserves the original error
not ok 11 - a failed request cancels its still-pending partner and preserves the original error
  ---
  duration_ms: 4.090585
  location: '/tmp/rentalcore-terminradar-negative-lq0h_xud/tests/dashboard.test.mjs:146:1'
  failureType: 'testCodeFailure'
  error: |-
    Expected values to be strictly equal:

    false !== true

  code: 'ERR_ASSERTION'
  name: 'AssertionError'
  expected: true
  actual: false
  operator: 'strictEqual'
  stack: |-
    TestContext.<anonymous> (file:///tmp/rentalcore-terminradar-negative-lq0h_xud/tests/dashboard.test.mjs:164:12)
    async Test.run (node:internal/test_runner/test:797:9)
    async Test.processPendingSubtests (node:internal/test_runner/test:526:7)
  ...
# Subtest: cleanup cancels both pending data requests
ok 12 - cleanup cancels both pending data requests
  ---
  duration_ms: 1.143358
  ...
1..12
# tests 12
# suites 0
# pass 11
# fail 1
# cancelled 0
# skipped 0
# todo 0
# duration_ms 2032.617405
```
