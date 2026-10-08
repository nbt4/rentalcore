# Prüfprotokoll Mietprodukt-Korrektur

Stand: 2026-10-08. Isolierter Worktree `fix/rental-product-logic`.
Lokale PostgreSQL 16 mit eigenem Test-Volume; nur erfundene Daten.
Produktive Daten wurden nicht abgefragt oder geändert. Kein Release oder PR.

Die verbindliche RentalCore-Gate-Reihenfolge wurde eingehalten. Der Nutzer hat
am Formatgate ausdrücklich erlaubt, den vorhandenen Bestandsfehler zu
protokollieren und die weiteren Gates auszuführen. Die geänderten Go-Dateien
sind vollständig formatiert. Unberührte Dateien wurden nicht umformatiert.

## Formatgate: bekannte, vom Nutzer freigegebene Bestandsabweichung

Echte Ausgabe von `gofmt -l .`:

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

## Frontend-Lint

`cd web && npm run lint` — Exit 0. Bestehende Warnungen (zusammengefasst):

```text
web/src/pages/JobsPage.tsx
  996:32 warning React Hook useEffect has a missing dependency: 'load'
web/static/scanner/ui/ScannerView.tsx
  216:8 warning useCallback: setupEventListeners, startScanning
  239:8 warning useCallback: handleCameraError, handleCameraFrame, handleDecodeResult,
        handleDecoderError, handleDoubleTapGesture, handleFocusGesture, handleZoomGesture
  264:8 warning useCallback: showScanFeedback
  288:8 warning useCallback: calculateROI
5 problems (0 errors, 5 warnings)
```

## Frontend-Build

`cd web && npm run build` — Exit 0. Echte Ausgabe des Build-Ergebnisses:

```text
vite v7.3.1 building client environment for production...
✓ 1766 modules transformed.
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-HNnxUf-n.js   489.47 kB │ gzip: 145.51 kB
✓ built in 7.74s
```

Bestehende Browserslist-Daten sind sieben Monate alt; kein Dependency-Update
vorgenommen. Die generierte, eingecheckte dist/index.html wurde nach dem Build
auf den unveränderten Ausgangszustand zurückgesetzt.

## Go-Build

`make build` — Exit 0:

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

## Vollständige Go-Tests, ohne Cache

`RENTALCORE_TEST_POSTGRES_DSN=<lokale _test-Datenbank> go test -count=1 ./...`
— Exit 0. Echte Ausgabe:

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.039s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.009s
ok  	go-barcode-webapp/internal/handlers	7.171s
ok  	go-barcode-webapp/internal/jev	0.028s
ok  	go-barcode-webapp/internal/jobstatus	0.007s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.134s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.712s
ok  	go-barcode-webapp/internal/services	0.024s
ok  	go-barcode-webapp/internal/services/pdf	0.028s
ok  	go-barcode-webapp/internal/services/postalcode	0.012s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.034s
?   	go-barcode-webapp/tools	[no test files]
```

Der neue PostgreSQL-Integrationstest führt die echten Positions-/Kosten-
Migrationen aus, prüft Wiederholbarkeit ohne Datenänderung und führt API-Anlage,
MCP-Vorschau/Anlage/Replay/Repair, native Reads, tatsächliche Analytics,
Mengenänderungen, Tagesmodus und Archivierung aus. Rechte, fehlende Preise,
Duplikate, Audit-Rollback und veraltete Vorschauen werden negativ geprüft.
JOB001165 ist kein Testdatensatz; der Repair-Test verwendet eine erfundene ID.

## Vet

`go vet ./...` — Exit 0, leere Ausgabe.

## UI-Payloadtests

`node --test web/tests/rental-position.test.mjs` (vorhandenes Node 24 mit TypeScript-Direktimport) — Exit 0. Echte Ausgabe:

```text
✔ + Mietprodukt builds a real rental line at the customer price (2.940219ms)
✔ missing prices block instead of inventing a price; explicit zero remains valid (1.283691ms)
ℹ tests 2
ℹ suites 0
ℹ pass 2
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 324.343625
```

Die geänderte Oberfläche ist die React-SPA `JobPositionsPanel.tsx`. Der
Anlagepayload und die normalen Job-/Analytics-APIs wurden automatisiert geprüft.
Ein manueller Browser-Klicktest oder Screenshots wurden nicht durchgeführt;
die Tabellen-/Layoutprimitives wurden nicht geändert.
