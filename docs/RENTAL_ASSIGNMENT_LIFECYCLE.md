# Fremdmiet-Zuweisungen: Ändern, Archivieren und Wiederherstellen

RentalCore **5.3.128** und Cores MCP **1.5.67**. Der Katalog erhält neun neue
Werkzeuge und umfasst damit **464 Tools: 118 Abfragen, 173 Vorschauen und
173 Ausführungen**. Bestehende Create-/Repair- und Katalogaktionen bleiben erhalten.

## Werkzeuge und Auflösung

`rental.job_external_equipment.get`, `.search` und `.audit_history` lesen
Zuweisungen und deren Historie. `search` benötigt die genaue Job-ID und liefert
standardmäßig aktive Zuweisungen; `include_archived=true` findet auch die
archivierten Positions-IDs und Versionen für Restore. `get` benötigt die genaue
Job-/Equipment-ID und liest beide Zustände. Private Notizen und rohes Audit-JSON
werden nicht ausgegeben. Finanzielle Get/Search-Abfragen benötigen aktuelle
DB-Adminrechte und separat `cores:rental:financial`; die minimale History braucht
Adminrechte und enthält keine Geldwerte, Notizen oder privaten Inhalte.

Die neuen Prepare/Execute-Paare sind `prepare_update/update`,
`prepare_archive/archive` und `prepare_restore/restore`. Namen tragen den Präfix
`rental.job_external_equipment.`; Clients können Punkte als Unterstriche zeigen.
Die allgemeine Katalogpflege bleibt unter `rental.external_equipment.*`.

## Änderbare Felder

| Feld | Wirkung und Grenzen |
| --- | --- |
| `quantity` | Ganze Menge 1–1000; Position und Lieferantenledger bleiben synchron. |
| `days_used` | Einkaufsmietdauer 1–365 Tage. Der Jobschalter `multiply_by_days` entscheidet über den Kostenfaktor. |
| `rental_unit_price` | Expliziter neuer Einkaufspreis-Snapshot in EUR, nichtnegativ, höchstens zwei Nachkommastellen bis 9999999999,99. Weglassen erhält den gespeicherten NUMERIC-Wert exakt, auch historisch zusätzliche Dezimalstellen. |
| `customer_price` | Expliziter Kunden-Stückpreis der Position, gleiche Geldgrenzen. Weglassen erhält den gespeicherten Preis. |
| `description` | Positionsbeschreibung bis 10000 Zeichen; leer löscht den Text. |
| `notes` | Zuweisungsnotiz bis 500 Bytes; leer löscht den Text. |
| `unit` | Nichtleere Einheit bis 50 Zeichen. |
| `follow_day_factor` | Kaufmännischer Folgetagfaktor 0–99,99, höchstens zwei Nachkommastellen. Unabhängig von den Einkaufsmiettagen. |
| `discount_percent`, `tax_rate` | Positionsrabatt und Steuersatz 0–100 mit höchstens zwei Nachkommastellen. |
| `discount_amount` | Nichtnegativer Positionsrabatt in EUR mit höchstens zwei Nachkommastellen. |
| `sort_order` | Nichtnegative Sortierung bis 2147483647. |

Weggelassene Geschäftsfelder bleiben erhalten. Explizite Nullpreise sind erlaubt;
JSON-null ist kein Löschbefehl. `total_cost`, Identität und Fremdschlüssel sind
nicht frei beschreibbar. Die JSON-Anfrage ist auf 256 KiB begrenzt.

Die Vorschau zeigt Ist/Soll, Diff, aktive Bearbeiter, Katalogstatus und Änderungen
an Lieferantenkosten, Netto-/Bruttoumsatz, erwarteter Nettomarge und Joberlös.
Sie verwendet die bestehende RentalCore-Logik für Umsatzsteuer, Positions-/
Jobrabatte und Eventtage. Eventtage sind die Differenz Enddatum minus Startdatum,
mit mindestens einem Tag. Neue Katalogpreise werden nie still auf vorhandene
Zuweisungen übernommen; neue Stückpreise müssen ausdrücklich im Entwurf stehen.

## Bestätigter Ablauf und Rechte

1. Job auflösen, dann Zuweisung inklusive `position_id` über Search/Get bestimmen.
2. Passendes `prepare_*` mit genauem `job_id`, `equipment_id`, `position_id` und
   gegebenenfalls Teiländerung aufrufen. Finale Felder, Diff und Wirkungen prüfen.
3. Fachliche Bestätigung einholen. Ausführen mit unverändertem Entwurf,
   `expected_updated_at` (Ledger), `expected_position_updated_at`,
   `expected_job_updated_at`, `expected_context`, geliefertem
   `required_confirmation_text` als `confirmation_text`, `confirm_change=true`
   und eigenem `idempotency_key`. `dry_run=true` bleibt reine Vorschau.
4. Bei HTTP409, veralteter Version oder verändertem Kontext neu vorbereiten.

Alle Vorbereitungen, Ausführungen und Erfolgswiederholungen verlangen einen
realen aktiven Rental-Administrator, das ausgewählte Aktionsrecht und unabhängig
davon `cores:rental:financial`. Update verwendet `cores:rental:update`, Archive
und Restore `cores:rental:archive`; Legacy `cores:write` bleibt unterstützt.
Das MCP veröffentlicht Scopes und OAuth-Challenges. Der Owner prüft die aktuellen
DB-Rechte vor jedem gespeicherten Beleg erneut. Identisches Request/Key liefert
die ursprüngliche Antwort; andere Felder mit gleichem Key werden verweigert.

## Erhaltene Historie und Grenzen

Archivieren erhält Positions-ID, Geschäfts-/Preisfelder, verknüpftes Ledger und
Kosten-Snapshots. Nur die aktive Umsatz-/Kostenwirkung entfällt; beim letzten
Archiv wird der Joberlös ausdrücklich auf null neu berechnet. Archivierte Kosten
werden bei späteren Job-Tagesänderungen oder Timestamp-Updates nicht verändert.
Restore erhält die gespeicherten Stückpreise und Geschäftsfelder. Der aktuelle
Job-Tagesfaktor wird in der Vorschau neu angewendet und der Kostenbetrag beim
Restore normalisiert; eine solche Kostenänderung steht ausdrücklich im Entwurf.

Ein archivierter Job muss zuerst wiederhergestellt werden; ein archivierter
Katalogartikel ebenfalls vor Update/Restore. Aktive Bearbeiter, angehängte
Positionsgeräte oder eine weitere aktive Position für dieselbe Job-/Equipment-
Identität blockieren die Aktion. Archive/Restore erlauben keine gleichzeitigen
Geschäftsfeldänderungen. Andere Positionstypen erhalten dadurch keinen Restore.

Ältere unverkettete Kostenzeilen benötigen zuerst den bereits vorhandenen
expliziten `repair_existing`-Ablauf von Prepare/Create. Neue Create-Aufrufe dürfen
archivierte, erhaltene Zuweisungen nicht ersetzen. Harte Ledgerlöschung,
Umschreiben der Identität oder Bearbeiten archivierter Snapshots sind verboten.
Native Legacy-/PDF-Schreibwege dürfen verknüpfte Positionen nicht überschreiben;
die native Jobansicht blendet archivierte Kosten aus. Native kommerzielle Anlage,
Änderung und Archivierung erhalten tatsächliches Vorher/Nachher-Audit und
Jobhistorie atomar. Native PUT lädt unter Joblock erneut und verweigert Änderungen
seit dem ursprünglichen Lesen, einschließlich eines zwischenzeitlichen Restore.
Delegierte MCP-Sitzungen müssen die benannten Owner-Verträge verwenden und dürfen
Positions-/Settings-, Legacy-Jobformular- oder PDF-Finalize-Wege nicht umgehen.

Position, Kosten, neuberechneter Joberlös/-version, Audit, native Jobhistorie und
dauerhafter Erfolgsbeleg committen gemeinsam. Fehler rollen die Transaktion
zurück. Keine Bestände, Gerätebewegungen, Bestellungen oder Lieferantennachrichten
werden durch diese Aktionen verändert.

## Schema und Veröffentlichung

Root/Rental-Migration **053** enthält identische additive Funktionen/Trigger und
wird bei Rental-Startup nach den bisherigen Positions-/Kostenregeln installiert.
Es gibt keine neuen Spalten, keine Bestandsdatenumschreibung und keine neue
Runtime-Variable. Reapplication ist idempotent; der Warehouse-Startup behält die
neuen Regeln, da dessen Katalogschema diese Funktionen nicht überschreibt.

RentalCore zuerst ausrollen, danach MCP. Images werden nach Prüfung der zuletzt
veröffentlichten Tags aus committed Quellcode als neue Version und `latest` mit
OCI-Revision gebaut. Submodule, beide Compose-Dateien und Dienstinventar laufen
synchron mit dem veröffentlichten Stack. Laufzeitumgebung, Volumes, Netzwerke,
Ports und andere Dienste bleiben erhalten; Health und Revision werden geprüft.
MCP-Clients müssen den Werkzeugkatalog aktualisieren und nötige Aktions-/
Finanzrechte separat über OAuth freigeben. Keine produktiven Fachmutationen oder
manuellen Migrationen als Release-Test. Ein Image-Rollback entfernt die additiven
Guards nicht und muss ihre Kompatibilität berücksichtigen.

Personal-Zuordnungen sowie weitere Planner-/Dokument-/Finanz-/Undo-/Legacy-
Abnahmen bleiben in [#4](https://github.com/nbt4/cores-mcp/issues/4) und
[#5](https://github.com/nbt4/cores-mcp/issues/5) offen.

Echte Ausgaben: [Prüfprotokoll](RENTAL_ASSIGNMENT_LIFECYCLE_TESTS.md).
