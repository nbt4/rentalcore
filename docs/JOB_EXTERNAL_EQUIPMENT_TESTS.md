# Prüfprotokoll: Mietprodukte am Job

Stand: 2026-10-07. Entwicklung in separaten Worktrees auf
`feat/mcp-job-external-equipment`. Keine neue Abhängigkeit oder Migration.

RentalCore nutzt PostgreSQL 16 in einem eigens gestarteten lokalen Testcontainer
mit eigenem Volume und synthetischen Daten. Die Integrationsprüfungen laufen
mit `RENTALCORE_TEST_POSTGRES_DSN`; Cores-MCPs Datenbankprüfungen mit den
bestehenden `CORES_MCP_*_TEST_DATABASE_URL`-Variablen. Verbindung und Zugangsdaten
werden hier nicht veröffentlicht. Alle Go-Testläufe verwenden `-count=1`.

RentalCore-Gates wurden der Reihe nach ausgeführt:

1. `gofmt -l .`: 43 vorhandene Dateien im Ausgangsstand, unverändert dieselbe
   Liste nach der Änderung. Der Nutzer hat die Fortsetzung mit dieser Ausnahme
   ausdrücklich erlaubt. Sämtliche neu angelegten/geänderten Go-Dateien sind
   formatiert. Die Ausgabe folgt unten.
2. `cd web && npm run lint`: Exit 0, keine Fehler, fünf sichtbare vorhandene
   Hook-Warnungen. Die fehlende ESLint-Konfiguration wurde auf Nutzerwunsch
   ergänzt. Empfohlene JS-/TypeScript-/Hook-/Refresh-Regeln bleiben aktiv;
   aufgedeckte Fehler werden durch Typen, Fehlerbehandlung und Trennung des
   Auth-Hooks vom Provider behoben. Der Scanner wird ebenfalls weiter geprüft.
   Es wurden keine Lintregeln abgeschaltet. Die Änderungen an den Frontends
   betreffen Typen/Modulimporte und keine sichtbaren Bedienabläufe.
3. `cd web && npm run build`: Exit 0, TypeScript und Vite erfolgreich.
4. `make build`: Exit 0.
5. `go test -p 1 -count=1 ./...`: Exit 0 mit lokalem PostgreSQL; alle Pakete grün.
6. `go vet ./...`: Exit 0, keine Ausgabe.

Cores-MCP: Der vollständige Lauf `go test -p 1 -count=1 ./...` mit sämtlichen
PostgreSQL-Testvariablen ist grün. Die neue Referenzauflösung und alle bestehenden
Datenbankprüfungen einschließlich `TestFlexibleQueryEntitiesAgainstDatabase`
wurden ausgeführt. `go vet ./...` und
`go build -o /tmp/cores-mcp-external-equipment-server ./cmd/server` sind ebenfalls
mit Exit 0 erfolgreich; beide erzeugen keine Ausgabe. `gofmt -l .` und
`git diff --check` melden für Cores-MCP keine Abweichungen.

Die lokale Testdatenbank erhielt zunächst alle Umbrella-Migrationen. Für das
vollständige Suite-Schema wurden zusätzlich die vorhandenen WarehouseCore-
Migrationen 039 (Lagerabläufe/Cases), 040 (Gerätezustand) und 044
(`core_product_links`) sowie die 91 unverändert übernommenen SQL-Anweisungen
von `EnsureProductMasterSchema` eingespielt. Dazu gehören auch
`products.product_code` und die zugehörigen Defaults/Indizes/Identifier.
Die fehlenden Strukturen wurden in der Testumgebung behoben. Repository-
Migrationen, Tests und deren Anforderungen wurden dafür nicht geändert.

Die neue RentalCore-Integration prüft Vorschau ohne Schreibwirkung, Mengen- und
Tagesgrenzen, fehlende Preise, ausdrücklich gespeicherte Nullpreise, Tagesfaktor,
veraltete Vorschauen, Archivzustand, aktive Bearbeiter, Duplikate, gebundene
Bestätigung, unveränderte Umsätze/Materialpositionen, Rechteentzug vor Replay,
Schlüsselkonflikte sowie Rollback von Zuordnung, Jobversion und Beleg bei einem
Fehler im finalen Audit. MCP-Tests prüfen zudem, dass Mehrdeutigkeit weder Job
noch Mietprodukt automatisch auswählt und Wiederholungen den Eigentümer erreichen.

Die Entwicklung und Prüfung sind abgeschlossen. Die 43 vorhandenen RentalCore-
Formatabweichungen bleiben mit der ausdrücklich erteilten Nutzer-Ausnahme
bestehen; der Frontend-Lint hat keine Fehler und fünf vorhandene Hook-Warnungen.
Die Änderung wird als Entwurfs-PR für Release 5.3.122 vorgelegt. Die Dienste
werden erst nach menschlichem Merge veröffentlicht; produktive Aufträge
werden bei der Release-Prüfung nicht geändert.

## RentalCore Format

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

## RentalCore Frontend-Lint

```text

> rentalcore-web@0.0.0 lint
> eslint .


web/src/pages/JobsPage.tsx
  996:32  warning  React Hook useEffect has a missing dependency: 'load'. Either include it or remove the dependency array  react-hooks/exhaustive-deps

web/static/scanner/ui/ScannerView.tsx
  216:8  warning  React Hook useCallback has missing dependencies: 'setupEventListeners' and 'startScanning'. Either include them or remove the dependency array                                                                                                                      react-hooks/exhaustive-deps
  239:8  warning  React Hook useCallback has missing dependencies: 'handleCameraError', 'handleCameraFrame', 'handleDecodeResult', 'handleDecoderError', 'handleDoubleTapGesture', 'handleFocusGesture', and 'handleZoomGesture'. Either include them or remove the dependency array  react-hooks/exhaustive-deps
  264:8  warning  React Hook useCallback has a missing dependency: 'showScanFeedback'. Either include it or remove the dependency array                                                                                                                                               react-hooks/exhaustive-deps
  288:8  warning  React Hook useCallback has a missing dependency: 'calculateROI'. Either include it or remove the dependency array                                                                                                                                                   react-hooks/exhaustive-deps

✖ 5 problems (0 errors, 5 warnings)
```

## RentalCore Frontend-Build

```text

> rentalcore-web@0.0.0 build
> tsc -b && vite build

vite v7.3.1 building client environment for production...
transforming...
Browserslist: browsers data (caniuse-lite) is 7 months old. Please run:
  npx update-browserslist-db@latest
  Why you should do it regularly: https://github.com/browserslist/update-db#readme
✓ 1765 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-fuMYYBw2.js   489.09 kB │ gzip: 145.37 kB
✓ built in 8.23s
```

## RentalCore Go-Build

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

## RentalCore Tests mit PostgreSQL

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.031s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.009s
ok  	go-barcode-webapp/internal/handlers	6.347s
ok  	go-barcode-webapp/internal/jev	0.021s
ok  	go-barcode-webapp/internal/jobstatus	0.008s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.148s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.798s
ok  	go-barcode-webapp/internal/services	0.025s
ok  	go-barcode-webapp/internal/services/pdf	0.026s
ok  	go-barcode-webapp/internal/services/postalcode	0.013s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.036s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Cores-MCP vollständiger PostgreSQL-Lauf

```text
ok  	github.com/nbt4/cores-mcp/cmd/server	0.011s
ok  	github.com/nbt4/cores-mcp/cmd/smoke	0.010s
ok  	github.com/nbt4/cores-mcp/internal/authn	0.056s
ok  	github.com/nbt4/cores-mcp/internal/config	0.009s
ok  	github.com/nbt4/cores-mcp/internal/httpx	0.009s
ok  	github.com/nbt4/cores-mcp/internal/mcpserver	12.005s
?   	github.com/nbt4/cores-mcp/internal/store	[no test files]
```

## Release-Gates für RentalCore 5.3.122

Die Gates wurden in der vorgeschriebenen Reihenfolge erneut ausgeführt.
Die Formatliste enthält unverändert dieselben 43 vom Nutzer als Ausnahme
freigegebenen Altdateien. Der Frontend-Lint hat keine Fehler und fünf vorhandene
Warnungen. Der Go-Testlauf verwendet `-count=1` und die lokale PostgreSQL-Testdatenbank.

### Frontend-Lint

```text

> rentalcore-web@0.0.0 lint
> eslint .


web/src/pages/JobsPage.tsx
  996:32  warning  React Hook useEffect has a missing dependency: 'load'. Either include it or remove the dependency array  react-hooks/exhaustive-deps

web/static/scanner/ui/ScannerView.tsx
  216:8  warning  React Hook useCallback has missing dependencies: 'setupEventListeners' and 'startScanning'. Either include them or remove the dependency array                                                                                                                      react-hooks/exhaustive-deps
  239:8  warning  React Hook useCallback has missing dependencies: 'handleCameraError', 'handleCameraFrame', 'handleDecodeResult', 'handleDecoderError', 'handleDoubleTapGesture', 'handleFocusGesture', and 'handleZoomGesture'. Either include them or remove the dependency array  react-hooks/exhaustive-deps
  264:8  warning  React Hook useCallback has a missing dependency: 'showScanFeedback'. Either include it or remove the dependency array                                                                                                                                               react-hooks/exhaustive-deps
  288:8  warning  React Hook useCallback has a missing dependency: 'calculateROI'. Either include it or remove the dependency array                                                                                                                                                   react-hooks/exhaustive-deps

✖ 5 problems (0 errors, 5 warnings)
```

### Frontend-Build

```text

> rentalcore-web@0.0.0 build
> tsc -b && vite build

vite v7.3.1 building client environment for production...
transforming...
Browserslist: browsers data (caniuse-lite) is 7 months old. Please run:
  npx update-browserslist-db@latest
  Why you should do it regularly: https://github.com/browserslist/update-db#readme
✓ 1765 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   1.64 kB │ gzip:   0.72 kB
dist/assets/index-CAnzfy0W.css   89.45 kB │ gzip:  16.59 kB
dist/assets/index-fuMYYBw2.js   489.09 kB │ gzip: 145.37 kB
✓ built in 8.70s
```

### make build

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

### Go-Tests mit PostgreSQL

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.032s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.010s
ok  	go-barcode-webapp/internal/handlers	7.897s
ok  	go-barcode-webapp/internal/jev	0.018s
ok  	go-barcode-webapp/internal/jobstatus	0.006s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.143s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	0.991s
ok  	go-barcode-webapp/internal/services	0.021s
ok  	go-barcode-webapp/internal/services/pdf	0.023s
ok  	go-barcode-webapp/internal/services/postalcode	0.014s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.038s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

### go vet

```text
Keine Ausgabe; Exit 0.
```
