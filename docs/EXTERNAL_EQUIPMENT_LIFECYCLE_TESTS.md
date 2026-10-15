# Fremdmietkatalog-Lebenszyklus: Prüfprotokoll

2026-10-15. Ausschließlich synthetische Daten in lokalem PostgreSQL16,
Datenbank `cores_catalog_lifecycle_test`, eigener Compose-Stack und frisches
Volume mit kanonischen Root-Migrationen. Kein produktiver Fachdatentest.

## Vollständige Go-Prüfung

`go test -count=1 ./...` mit den passenden isolierten Testdatenbankvariablen.

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.035s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.013s
ok  	go-barcode-webapp/internal/handlers	22.277s
ok  	go-barcode-webapp/internal/jev	0.062s
ok  	go-barcode-webapp/internal/jobstatus	0.014s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.380s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	1.539s
ok  	go-barcode-webapp/internal/services	0.023s
ok  	go-barcode-webapp/internal/services/pdf	0.037s
ok  	go-barcode-webapp/internal/services/postalcode	0.011s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.024s
?   	go-barcode-webapp/tools	[no test files]
```

## Race

`go test -race -count=1 ./internal/handlers ./internal/schema ./internal/repository` mit den passenden isolierten Testdatenbankvariablen.

```text
ok  	go-barcode-webapp/internal/handlers	24.552s
ok  	go-barcode-webapp/internal/schema	3.028s
ok  	go-barcode-webapp/internal/repository	1.570s
```

`go vet ./...` und Serverbuild erfolgreich. Frontendlint:0Fehler,5bestehendeWarnungen. `make build` erfolgreich.

Die Tests prüfen Teiländerungen, Nullpreise, exakte Version/Kontextbindung,
Archivierung/Wiederherstellung, offene Jobreferenzen aus beiden Tabellen,
historische Preissnapshots, Zusatzfelder, Lieferanten/Duplikate, Rollback bei
Auditfehlern, dauerhaftes Replay und Rechteentzug. Rental führt echte Zwei-
Verbindungs-Rennen aus: Archivierung gegen beide Referenzquellen/Reopen sowie
Reopen ohne bisherige Referenzen gegen die erste neue Referenz, jeweils beide
Reihenfolgen. Die bestehende echte Child-Versionstrigger-Kette wird gegen den
Owner-Sperrplan geprüft. Schema-Reapplication ist idempotent; SQL-Kopien stimmen.

Unabhängiges Review abgeschlossen ohne verbleibende konkrete Findings.
Gefundene MVCC-Reopen-Lücke, umgekehrte Child-/Parent-Sperrreihenfolge und
Native-Editor-Zusatzfeld-Randfälle behoben und mit Regressionen gesichert.
Suite-CI führt die neuen Owner-/Native-/MCP-PostgreSQL-Tests mit Race aus.

Frontend: `npm ci` → `npm run build` erfolgreich. Keine UI-Quelländerung.
