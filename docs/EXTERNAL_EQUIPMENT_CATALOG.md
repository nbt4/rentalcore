# Fremdmietkatalog über MCP

RentalCore 5.3.126 / Cores MCP 1.5.63 ergänzen
`rental.external_equipment.prepare_create` und `rental.external_equipment.create`.
MCP-Clients können diese Namen als `rental_external_equipment_prepare_create`
und `rental_external_equipment_create` darstellen.

Die Anlage schreibt ausschließlich einen aktiven Datensatz in den allgemeinen
Fremdmietkatalog `rental_equipment`. Sie legt weder einen Warehouse-Artikel noch
Geräte, Lagerbestand, Bestellungen oder Jobzuordnungen an. Zur späteren
Jobzuordnung dient das bestehende Paar `rental.job_external_equipment.prepare_create/create`.

## Felder

| Feld | Bedeutung |
| --- | --- |
| `name` | Bezeichnung, Pflicht, maximal 255 Zeichen |
| `category` | Kategoriebezeichnung, Pflicht, maximal 100 Zeichen |
| `supplier` | Lieferantenname, maximal 255 Zeichen; Pflicht ohne `supplier_id` |
| `supplier_id` | Optional: exakte ID eines aktiven Rental-Lieferantenkontakts; kein Procurement-Lieferant. Ein zusätzlich angegebener Name muss passen. |
| `rental_price` | Einkaufsmietpreis des Lieferanten in EUR |
| `customer_price` | Kundenmietpreis in EUR; bei Jobzuordnung gelten die bestehenden Preis-/Steuer-/Tagesregeln |
| `description`, `notes` | Optional, jeweils maximal 10000 Zeichen |
| `allow_similar` | Ähnliche Katalogeinträge nach bewusster Prüfung akzeptieren; hebt exakte Duplikatsperren nicht auf |

Beide Preise müssen ausdrücklich angegeben sein. `0` ist erlaubt, fehlende
Preise werden niemals als `0` übernommen. Zulässig sind nichtnegative Werte mit
höchstens zwei Nachkommastellen bis 99999999,99 EUR, passend zu NUMERIC(10,2).
Ein Lieferantenname darf ohne Kontakt-ID erfasst werden. Die Funktion legt
keinen Lieferantenkontakt automatisch an.

## Ablauf und Rechte

1. `prepare_create` mit allen Geschäftsfeldern aufrufen. Unvollständige Entwürfe
   nennen `required_fields`. Die Vorschau enthält alle Felder, mögliche Duplikate,
   Wirkungen, `expected_context` und `required_confirmation_text`.
2. Ähnliche Einträge prüfen und gegebenenfalls mit `allow_similar=true` erneut
   vorbereiten. Gleiche Bezeichnung und gleicher Lieferant werden unabhängig
   von Groß-/Kleinschreibung und äußeren Leerzeichen immer gesperrt, auch wenn
   der vorhandene Artikel inaktiv ist. Vorhandene Artikel verwenden.
3. Den vollständigen finalen Entwurf zeigen und fachliche Bestätigung einholen.
4. `create` mit unveränderten Geschäftsfeldern, exaktem `expected_context`,
   `confirmation_text=required_confirmation_text`, `confirm_change=true` und
   einem eigenen `idempotency_key` ausführen. `dry_run=true` erzwingt Vorschau.
5. Die zurückgegebene `equipment.equipment_id` für die gesondert bestätigte
   Jobzuordnung verwenden.

Vorschau und Ausführung benötigen einen echten angemeldeten Rental-Administrator,
`cores:rental:create` (oder Legacy-`cores:write`) und unabhängig davon
`cores:rental:financial`. Der MCP-Descriptor veröffentlicht diese Scopes samt
OAuth-Challenge für fehlende Freigaben. Aktuelle Zielrechte werden vor jedem
Replay erneut geprüft. Identischer Schlüssel/Payload liefert den gespeicherten
Erfolg; andere Felder mit demselben Schlüssel werden zurückgewiesen.

Der Owner prüft Lieferant, Duplikate und Vorschaukontext unter Datenbanksperren
erneut. Datensatz, Audit mit Ursprung `MCP/AI` und dauerhafter Wiederholungsbeleg
bilden eine Transaktion. Auditfehler rollen alles zurück. Es gibt keine neue
Migration, Abhängigkeit oder Umgebungsvariable. Antworten tragen den bestehenden
MCP-Zeitstempel und Quellnachweise.

## Beispiel einer Vorschau

```json
{
  "name": "Synthetisches Zumietprodukt",
  "category": "Audio",
  "supplier": "Beispiellieferant",
  "rental_price": 15.00,
  "customer_price": 25.00
}
```

## Release

RentalCore zuerst ausrollen, `/health` auf 5.3.126 prüfen, dann Cores MCP 1.5.63
und `/ready` prüfen. Der schreibfähige Katalog umfasst 446 Werkzeuge
(112 Abfragen, 167 Vorschauen, 167 Ausführungen). Bestehende MCP-Clients müssen
ihre Werkzeugdefinitionen per Refresh/Rescan laden; fehlende Create-/Finanzrechte
werden anschließend ausdrücklich über OAuth freigegeben. Produktive Katalog-
oder Jobanlagen sind kein Deployment-Test.

## Lokale Prüfung und Review

2026-10-15: Entwicklung in isolierten Worktrees aus GitHub `main`; fremde
Workspace-Änderungen wurden nicht übernommen. PostgreSQL 16 läuft ausschließlich
in einem eigenen lokalen Container mit synthetischen Daten und `_test`-Datenbank.
Die neue Owner-Integration prüft Pflichtfelder, explizite Nullpreise, Cent-/
Bereichsgrenzen, lange Unicode-Texte, aktive Lieferanten und Namenskonflikte,
reine Vorschauen, gebundene Bestätigung, Idempotenzkonflikte, aktuelle Rechte vor
Replay, inaktive/exakte Duplikate, veraltete Lieferanten-/Duplikatkontexte,
Audit-Rollback mit erfolgreichem Retry und zwei konkurrierende Verbindungen.
MCP prüft signierte Delegation, Dry-run, Retry/Replay, Rechte, Quellen/Zeitstempel,
Schema sowie echte HTTP-OAuth-Challenges und beide Descriptor-Felder.

Ein unabhängiger Review-Agent prüfte Rechte, Preise, SQL, Kontextbindung,
Duplikate und Katalogbegrenzung. Sein Befund zur zu kleinen Requestgrenze wurde
auf 256 KiB korrigiert und mit Unicode-Grenztexten getestet. Keine verbleibenden
blockierenden Findings. Die Race-Prüfung nutzt nun zwei DB-Verbindungen, damit
die Produktionssperre statt des Connection-Pools serialisiert.
