# Fremdmietkatalog: Ändern, Archivieren und Wiederherstellen

RentalCore 5.3.127, WarehouseCore 5.9.116 und Cores MCP 1.5.66.

Der allgemeine Fremdmietkatalog erhält neun zusätzliche Werkzeuge:
`rental.external_equipment.get`, `.search`, `.audit_history` sowie
`prepare_update/update`, `prepare_archive/archive`, `prepare_restore/restore`.
Die vorhandenen Create- und Jobzuweisungswerkzeuge bleiben erhalten.
MCP-Clients stellen Punkte gegebenenfalls als Unterstriche dar.
Der schreibfähige Katalog umfasst 455 Tools: 115 Abfragen, 170 Vorschauen und
170 Ausführungen. Clients müssen ihre Werkzeugdefinitionen aktualisieren.

## Felder und Ablauf

Update erlaubt ausschließlich `name`, `category`, `supplier`, `supplier_id`,
`rental_price`, `customer_price`, `description` und `notes`. Weggelassene Felder
bleiben erhalten. `supplier_id=0` entfernt nur die optionale Kontaktverknüpfung;
leere Beschreibung/Notiz löscht deren Inhalt. JSON-null ist kein Löschbefehl.
Preise sind nichtnegativ mit höchstens zwei Nachkommastellen bis 99999999,99 EUR;
Nullpreise sind ausdrücklich erlaubt. Identitätsfelder müssen ausgefüllt sein.
Ein verknüpfter Lieferant ist ein aktiver Rental-Lieferantenkontakt.

1. Exakte `equipment_id` auflösen; `search` liefert standardmäßig aktive Artikel,
   mit `include_inactive=true` auch archivierte. `get` liest beide Zustände.
2. Das passende `prepare_*` mit ID und gegebenenfalls Teiländerung aufrufen.
   Vollständigen aktuellen/finalen Datensatz, Diff, Lieferanten, ähnliche Artikel,
   Zusatzfelder, offene Jobs und Auswirkungen prüfen.
3. Finale fachliche Bestätigung einholen. Ausführen mit unverändertem Entwurf,
   exaktem `expected_updated_at`, `expected_context`, der gelieferten
   `required_confirmation_text` als `confirmation_text`, `confirm_change=true`
   und eigenem `idempotency_key`. `dry_run=true` bleibt Vorschau.
4. Bei veraltetem Kontext oder HTTP409 erneut vorbereiten. Nicht blind bestätigen.

Aktuelle aktive Rental-Administratorrechte und reale Nutzeridentität sind
verpflichtend. Update braucht `cores:rental:update`, Archivierung und
Wiederherstellung `cores:rental:archive` (Legacy `cores:write` wird weiterhin
unterstützt). Jede Vorbereitung, Ausführung und Erfolgswiederholung benötigt
zusätzlich separat erteiltes `cores:rental:financial`. Rechte werden am Owner
vor dem gespeicherten Beleg erneut geprüft. Gleiches Request/Key liefert die
ursprüngliche Antwort, geändertes Request mit gleichem Key wird abgelehnt.

## Fachliche Wirkung und Grenzen

Offene, nicht archivierte Jobs blockieren Archivierung sowie Änderung von
Bezeichnung oder Lieferantenidentität. Beide Referenzquellen zählen: echte
Rental-Positionen und historische, auch unverknüpfte Fremdmiet-Kostenzeilen.
Abgeschlossene/bezahlte/stornierte Jobs bleiben mit ihren Referenzen erhalten.
Neue offene Referenzen und Wiederöffnen/Wiederherstellen eines Jobs können
archivierte Katalogartikel nicht aktivieren. Gleichzeitige Vorgänge werden
über Datenbank-Sperren serialisiert; Kollisionen der Owner-API liefern409.
Ab mehr als1000 referenzierten Jobs ist eine Einzeländerung gesperrt.

Preisänderungen gelten für künftige Zuordnungen. Vorhandene Jobpreise,
Lieferantenkosten, Mengen, Miettage und Erlöse werden nicht neu berechnet.
Archivieren und Wiederherstellen erhalten ID, alle Geschäftsfelder, Preise und
Zusatzfelder. Archivierte Artikel müssen vor Bearbeitung wiederhergestellt
werden; ein archivierter verknüpfter Lieferant muss zuerst wieder aktiv sein.
Exakte normalisierte Name/Lieferant-Duplikate sind bei Anlage/Identitätswechsel
auch über archivierte Artikel hinweg gesperrt. Bestehende Altduplikate werden
nicht umgeschrieben und bleiben ohne Identitätswechsel bearbeitbar.

Die native Warehouse-DELETE-Route archiviert jetzt mit atomarem Audit, statt
physisch zu löschen. Wiederholtes Archivieren bleibt erfolgreich ohne doppeltes
Audit. Native Anlage/PUT/Archivierung committen mit ihrem Audit gemeinsam;
Zusatzfelder bleiben beim Zustandswechsel unverändert. Ein vollständiges
Editor-Payload mit identischen Zusatzwerten ist zulässig; echte Änderungen,
auch eine explizite leere Liste bei vorhandenen Werten, liefern409. Delegierte
MCP-Sitzungen dürfen weder native Katalog- noch Zusatzfeld-Schreibwege umgehen.

Neue minimale Lese-/History-Werkzeuge verlangen aktuelle DB-Adminrechte,
geben keine privaten Notizen oder rohes Audit-JSON aus und projizieren Preise
nur mit separatem Rental-Finanzscope. Die History vereinigt native und neue
Owner-Audits mit expliziten Feldern. Vorhandene allgemeine Lesewerkzeuge behalten
ihre Verträge. Zusatzfelder werden in der privilegierten Vorschau gebunden und
erhalten; ihre Bearbeitung ist kein neuer MCP-Vertrag dieses Releases.

Katalogänderung, tatsächliches Vorher/Nachher-Audit und dauerhafter Beleg bilden
eine Transaktion. Auditfehler rollen die gesamte Mutation zurück. Keine
Lagerartikel, Bestände, Bestellungen, Lieferantennachrichten oder Jobzuordnungen
werden durch diese Katalogaktionen angelegt oder verändert.

## Schema und Rollout

Additive Migration: Umbrella/Rental052, Warehouse065. Beide Owner installieren
identische Funktionen/Trigger beim Start mit begrenzter Sperr-/Laufzeit;
Wiederholung ist idempotent. Keine bestehenden Daten werden zurückgeschrieben,
keine neuen Pflichtvariablen und keine manuelle Produktionsmigration.
Zuerst RentalCore, danach WarehouseCore und MCP veröffentlichen/ausrollen.
Images werden aus gemergtem Quellcode mit Versions-/latest-Tag und OCI-Revision
gebaut; Dachrepository-Pins und Submodule laufen synchron mit dem Deployment.
Health/Readiness, Quellrevision und unveränderte Runtime-Inputs werden geprüft.
Produktive Fachmutationen sind kein Release-Test.

Jobzuweisungen selbst ändern/archivieren/wiederherstellen benötigen einen
eigenen Position-/Kostenledger-Vertrag und bleiben zusammen mit Personal,
Planner, Dokumenten, weiteren Finanz-Writer- und Undo-Prüfungen in Issues
[#4](https://github.com/nbt4/cores-mcp/issues/4) und
[#5](https://github.com/nbt4/cores-mcp/issues/5) offen.

Echte Testausgaben: [Prüfprotokoll](EXTERNAL_EQUIPMENT_LIFECYCLE_TESTS.md).
