# Rental-Finanzrechte: Prüfprotokoll

2026-10-17, isoliertes lokales PostgreSQL16 mit synthetischen Testdaten,
RENTALCORE_TEST_POSTGRES_DSN auf die dedizierte `_test`-Datenbank gesetzt.

Gezielte Tests prüfen Projektion, signed Cookie-/Bearer-Delegation vor
Sessionfallback, abgewiesene native GETs/History, HS256/384/512, falsche
Scope-/Financial-Claimtypen, aktuelle Nutzer-/Admin-/Aktivitätsbindung sowie
Null-/False-Finanzeingaben vor Datenbankzugriff. Der PostgreSQL-Jobtest prüft
vollständige Vorschau/Ausführung, operativen Replay nach Finanzrechteentzug
und unveränderte Originalbelege bei erneutem Grant. Material-/Staffing- und
Audit-Feldlisten wurden zusätzlich unabhängig geprüft.

## Gezielte Go-/PostgreSQL-Tests

```text
ok  	go-barcode-webapp/internal/handlers	2.149s
```

## go test -count=1 ./... -timeout 180s

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.036s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.008s
ok  	go-barcode-webapp/internal/handlers	85.047s
ok  	go-barcode-webapp/internal/jev	0.042s
ok  	go-barcode-webapp/internal/jobstatus	0.016s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.438s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	16.169s
ok  	go-barcode-webapp/internal/services	0.043s
ok  	go-barcode-webapp/internal/services/pdf	0.069s
ok  	go-barcode-webapp/internal/services/postalcode	0.007s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.035s
?   	go-barcode-webapp/tools	[no test files]
```

## go test -race -count=1 ./... -timeout 240s

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	1.159s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	1.066s
ok  	go-barcode-webapp/internal/handlers	65.636s
ok  	go-barcode-webapp/internal/jev	1.145s
ok  	go-barcode-webapp/internal/jobstatus	1.041s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	1.517s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	2.805s
ok  	go-barcode-webapp/internal/services	1.071s
ok  	go-barcode-webapp/internal/services/pdf	1.101s
ok  	go-barcode-webapp/internal/services/postalcode	1.038s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	1.155s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

go vet ./... und Serverbuild erfolgreich (Exit0, keine Ausgabe).
Frontend: npm ci und npm run build erfolgreich; Vite9.35s,1769Module.
Keine UI-Quelländerung. Umbrella-Designsystemprüfung erfolgreich.
Unabhängiges Abschlussreview ohne verbleibende P1/P2-Findings.
