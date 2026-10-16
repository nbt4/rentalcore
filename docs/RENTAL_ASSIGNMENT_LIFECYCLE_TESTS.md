# Fremdmiet-Zuweisungen: Prüfprotokoll

2026-10-16. Isolierter PostgreSQL16-Compose-Stack mit frischem Volume,
kanonischen Root-Migrationen einschließlich053 und ausschließlich synthetischen
Daten in `cores_assignment_lifecycle_test`. Keine produktiven Fachdatentests.

## Vollständige Go-Tests

`go test -count=1 ./...` mit den passenden isolierten Testdatenbankvariablen.

```text
?   	go-barcode-webapp/cmd/compliance	[no test files]
ok  	go-barcode-webapp/cmd/server	0.057s
?   	go-barcode-webapp/internal/cache	[no test files]
?   	go-barcode-webapp/internal/compliance	[no test files]
ok  	go-barcode-webapp/internal/config	0.012s
ok  	go-barcode-webapp/internal/handlers	42.845s
ok  	go-barcode-webapp/internal/jev	0.064s
ok  	go-barcode-webapp/internal/jobstatus	0.011s
?   	go-barcode-webapp/internal/logger	[no test files]
?   	go-barcode-webapp/internal/metrics	[no test files]
?   	go-barcode-webapp/internal/middleware	[no test files]
?   	go-barcode-webapp/internal/models	[no test files]
?   	go-barcode-webapp/internal/monitoring	[no test files]
ok  	go-barcode-webapp/internal/repository	0.233s
?   	go-barcode-webapp/internal/routes	[no test files]
?   	go-barcode-webapp/internal/scan	[no test files]
ok  	go-barcode-webapp/internal/schema	4.168s
ok  	go-barcode-webapp/internal/services	0.041s
ok  	go-barcode-webapp/internal/services/pdf	0.038s
ok  	go-barcode-webapp/internal/services/postalcode	0.019s
?   	go-barcode-webapp/internal/services/storage	[no test files]
?   	go-barcode-webapp/internal/services/warehousecore	[no test files]
ok  	go-barcode-webapp/internal/sync/m365	0.028s
?   	go-barcode-webapp/tools	[no test files]
?   	go-barcode-webapp/web/node_modules/flatted/golang/pkg/flatted	[no test files]
```

## Race

`go test -race -count=1 ./internal/handlers ./internal/schema ./internal/repository` mit den passenden isolierten Testdatenbankvariablen.

```text
ok  	go-barcode-webapp/internal/handlers	49.264s
ok  	go-barcode-webapp/internal/schema	2.885s
ok  	go-barcode-webapp/internal/repository	1.458s
```

`go vet ./...` und Serverbuild erfolgreich. Unabhängiges Code-/Dachrepo-
Review abgeschlossen ohne verbleibende konkrete Findings. Gefundene Wartezyklen
bei nativen Job-Zeilensperren, nicht atomarer Legacy-Linkprüfung, Unicode-
Requestgrenze und Create/Repair-Konfliktklassifikation behoben und abgesichert.

Die Regressionen prüfen erhaltene Einkaufs-/Kundenpreise trotz Katalogänderung,
Nullpreise, alle drei Versionen, reine Vorschau, gebundene Bestätigung,
Rechteentzug vor dauerhaftem Replay, Archiv-Freeze auch bei Job-Tageswechsel
und Timestamp-Update, explizite Restore-Neuberechnung, präzise NUMERIC-Snapshots,
letzten Joberlös null, Netto-/Bruttogoldens mit Job-/Positionsrabatt und Steuer,
10.000 Emoji, Editorkontext sowie native und MCP-Audit-Rollbacks.

Echte Zwei-Verbindungs-Rennen laufen mit den vorhandenen Job-/Child-Version-
Triggern: Archive gegen Mengenänderung in beiden Reihenfolgen, gehaltene native
Jobrow gegen Owner-Preview, Legacy-Writer gegen explizites Repair und alter
nativer HTTP-PUT über Preisänderung/Archive/Restore. Native delegierte Positions-,
Settings-, Jobformular- und PDF-Finalize-Aufrufe werden vor Datenbankzugriff
verweigert. Startup-/Standalone-/Root-SQL stimmen überein und sind wiederholbar.

MCP prüft signierte Delegation, Dry-run, Fehler/Retry/Replay, fehlende/falsche
Scopes und aktuelle Ownerrechte. Echte HTTP-MCP-Descriptor/Challenge-Tests
sowie DB-Adminentzug, archivierte Get/Search-Discovery und minimale finanzfreie
History-Projektion sind enthalten. Der Katalog umfasst464Tools.

Frontend: `npm ci`, `npm run lint` (0 Fehler,5 bestehende Warnungen) und `npm run build` erfolgreich. Keine UI-Quelländerung.
