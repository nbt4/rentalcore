# Prüfprotokoll: Fremdmietkatalog

2026-10-15. Alle Datenbankprüfungen nutzen ausschließlich einen lokalen
PostgreSQL-16-Testcontainer und synthetische Daten. Kein Produktionsdatensatz
wurde zum Test angelegt oder geändert. Alle Befehle endeten mit Exit 0.

Reihenfolge: geänderte Go-Dateien mit gofmt/goimports formatieren; Suite-Designprüfung;
Frontend-Lint und -Build; make build; vollständige Go-Tests mit -count=1; Vet.
Zusätzlich gezielte Race-Integration. Unveränderte 43 Go-Formatabweichungen und
fünf Frontend-Hook-Warnungen sind im vorhandenen Releaseprotokoll dokumentiert.

## make build

```text
Building TS Jobscanner server...
go build -o server cmd/server/main.go
```

## go test -count=1 ./... (RENTALCORE_TEST_POSTGRES_DSN gesetzt)

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.029s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.011s
ok  	go-barcode-webapp/internal/handlers	18.832s
ok  	go-barcode-webapp/internal/jev	0.051s
ok  	go-barcode-webapp/internal/jobstatus	0.009s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.681s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	2.937s
ok  	go-barcode-webapp/internal/services	0.024s
ok  	go-barcode-webapp/internal/services/pdf	0.030s
ok  	go-barcode-webapp/internal/services/postalcode	0.010s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.033s
?   	go-barcode-webapp/tools	[no test files]
```

## go vet ./...

```text
(keine Ausgabe)
```

## go test -race -count=1 ./internal/handlers -run TestRentalExternalCatalog -v

```text
=== RUN   TestRentalExternalCatalogAtomicCreateAndGuards
--- PASS: TestRentalExternalCatalogAtomicCreateAndGuards (0.78s)
PASS
ok  	go-barcode-webapp/internal/handlers	1.894s
```

## Frontend lint/build

```text
> rentalcore-web@0.0.0 lint
> eslint .


/app/src/pages/JobsPage.tsx
  996:32  warning  React Hook useEffect has a missing dependency: 'load'. Either include it or remove the dependency array  react-hooks/exhaustive-deps

/app/static/scanner/ui/ScannerView.tsx
  216:8  warning  React Hook useCallback has missing dependencies: 'setupEventListeners' and 'startScanning'. Either include them or remove the dependency array                                                                                                                      react-hooks/exhaustive-deps
  239:8  warning  React Hook useCallback has missing dependencies: 'handleCameraError', 'handleCameraFrame', 'handleDecodeResult', 'handleDecoderError', 'handleDoubleTapGesture', 'handleFocusGesture', and 'handleZoomGesture'. Either include them or remove the dependency array  react-hooks/exhaustive-deps
  264:8  warning  React Hook useCallback has a missing dependency: 'showScanFeedback'. Either include it or remove the dependency array                                                                                                                                               react-hooks/exhaustive-deps
  288:8  warning  React Hook useCallback has a missing dependency: 'calculateROI'. Either include it or remove the dependency array                                                                                                                                                   react-hooks/exhaustive-deps

✖ 5 problems (0 errors, 5 warnings)


> rentalcore-web@0.0.0 build
> tsc -b && vite build

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
✓ built in 12.04s
```

## Diff und Designsystem

```text
git diff --check: Exit 0, keine Ausgabe
Geänderte Go-Dateien: gofmt/goimports, keine verbleibende Ausgabe
Designsystem-Prüfung erfolgreich.
```
