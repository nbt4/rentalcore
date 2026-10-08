# Mietprodukte: Auftragsposition und Lieferantenkosten

`job_positions` mit `position_type='rental'` und `rental_equipment_id` ist die
kanonische kaufmännische Mietprodukt-Position. Job-UI, Positions-Read und
Auftragswert lesen diese Positionen. `rental_equipment` bleibt der bestehende
Katalog; es entstehen keine Warehouse-Produkte oder Bestandsanforderungen.

## Preise und Berechnung

- `rental_equipment.rental_price` ist der Lieferanten-/Einkaufspreis.
- `rental_equipment.customer_price` ist der Kundenpreis. Eine neue Position
  übernimmt ihn als `job_positions.unit_price`, auch wenn ein API-Client einen
  anderen Preis mitsendet. Spätere ausdrücklich vorgenommene Positionspreis-
  oder Rabattänderungen bleiben möglich.
- Beide Katalogpreise müssen vorhanden, nicht negativ und gültige Geldbeträge
  sein. NULL ist kein Preis von null Euro; ausdrücklich gespeicherte 0 ist erlaubt.
- `job_rental_equipment` bleibt ausschließlich der Lieferantenkostenbeleg.
  `position_id` verknüpft ihn eindeutig mit seiner Rental-Position;
  `rental_unit_price` bewahrt den bei der Anlage gespeicherten Einkaufspreis.
- Fremdmietkosten: gespeicherter Einkaufspreis × Menge × `days_used`, wenn
  `multiply_by_days=true`, sonst Einkaufspreis × Menge. Die UI verwendet ohne
  ausdrücklich angegebene `days_used` die bestehenden Veranstaltungstage.
- Die vorhandene Kundenpreislogik bleibt maßgeblich: Rental-Positionen starten
  wie bisher mit `follow_day_factor=0`, Steuer 19 %, ohne Positionsrabatt.
  Deshalb vervielfacht `days_used` nur die Lieferantenkosten, nicht automatisch
  den Verkaufspreis. Veranstaltungstage und ein später gesetzter Folgetagfaktor,
  Positions-/Jobrabatte und `prices_include_tax` folgen den vorhandenen Funktionen.
- Jobs speichern Umsatz brutto; Analytics weisen Netto und Brutto getrennt aus.
  Die Marge ist Nettoerlös nach Rabatten minus Fremdmietkosten. Kosten zählen
  genau einmal, auch bei mehreren historischen Positionen desselben Artikels,
  null Euro Kundenpreis oder vollständigem Rabatt.

Native UI-Anlage und MCP verwenden dieselbe transaktionale Anlagefunktion.
Position, Kostenbeleg und Neuberechnung werden zusammen gespeichert. Datenbank-
Trigger halten gespeicherte Kosten bei Mengenänderungen und beim Umschalten von
`multiply_by_days` konsistent, auch in anderen nativen Job-Schreibwegen. Eine
spätere Änderung des Katalog-Einkaufspreises ändert alte Kosten nicht.
Archivierte Positionen und ihre verknüpften Kosten verlassen aktive Auswertungen;
IDs, Position und Kostenbeleg bleiben erhalten. Eine vorhandene Zuordnung
blockiert eine weitere Anlage desselben Artikels im Job und verlangt eine
explizite Prüfung; sie wird nicht still überschrieben.

## MCP-Anlage

Die Namen `rental.job_external_equipment.prepare_create` und `.create` bleiben
stabil. Die Vorschau enthält Job, Artikel/Lieferant, Menge, Lieferanten-Miettage,
beide Katalogpreise, tatsächlichen Lieferanten-Einzelpreis, gesamte Kosten,
Kundenverkaufswert netto/brutto, erwartete Nettomarge und die Umsatzänderung.
Bei Altjobs mit manuell gespeichertem Umsatz kann die native Positions-
Neuberechnung diesen Wert ersetzen; die Vorschau zeigt den tatsächlichen
Vorher-/Nachher-Wert und zusätzlich den Beitrag der neuen Position.

Voraussetzungen: persönlicher Suite-Benutzer, aktuelle aktive Rental-Adminrechte,
Rental-Anlagerecht und gesonderter `cores:rental:financial`-Scope. Dry-run schreibt
nichts. Finale Ausführung verlangt exakte Jobversion, Kontextfingerprint,
bindende Bestätigungsphrase, `confirm_change=true` und `idempotency_key`.
Katalogpreise, sämtliche Jobpositionen, Kostenbelege und aktive Bearbeiter sind
an die Vorschau gebunden. Position, Kosten, Umsatz, Jobversion, Historie, Audit
und dauerhafter Replay-Beleg committen atomar. Identische Wiederholung liefert
das gespeicherte Ergebnis; andere Eingaben mit demselben Schlüssel blockieren.

`rental.jobs.get` liefert aktive `positions` und `rental_positions` sowie separat
`external_rentals` mit `position_id` und `repair_required`. Dieser finanzielle
Job-Read erfordert `cores:rental:financial` und veröffentlicht die OAuth-Scopes.

## Bestätigter Repair-Workflow

Es erfolgt keine automatische Datenreparatur und keine Änderung an JOB001165.
Nach menschlicher Freigabe kann eine einzelne bestehende Zuweisung so geprüft
und repariert werden:

1. `rental.jobs.get` für die Job-ID aufrufen, z. B. `1165`. Fehlende Positionen
   anhand der bestehenden `external_rentals` und `rental_positions` feststellen.
2. `prepare_create` mit exaktem `job_id`, `equipment_id`, der **gespeicherten**
   `quantity` und `days_used` sowie `repair_existing=true` aufrufen.
3. Die vollständige Vorschau prüfen. Sie zeigt den aktuellen Kundenpreis und
   die unveränderten historischen Gesamtkosten. Falls noch kein Einkaufspreis-
   Snapshot existiert, wird er aus den gespeicherten Kosten, Mengen und dem
   aktuellen Tagesmodus abgeleitet; der heutige Katalogpreis ersetzt ihn nicht.
4. Erst nach ausdrücklicher Zustimmung `.create` mit denselben Eingaben,
   finaler Version/Kontext/Bestätigungsphrase und einem **neuen** stabilen Schlüssel
   ausführen. Die Phrase beginnt im Repair-Modus mit `REPAIR`.

Repair erzeugt genau eine fehlende Position und ergänzt Link/Snapshot am
bestehenden Kostenbeleg. Menge, Miettage, Gesamtkosten, Notizen, Zeitstempel und
vorhandene Historie bleiben erhalten; neue Historie/Audit werden ergänzt.
Vorhandene Rental-Positionen, bereits verknüpfte Kosten, abweichende Menge/Tage,
fehlende Katalogpreise, widersprüchliche Kosten-Snapshots oder veraltete Vorschauen blockieren. Eine bestehende
korrekte Position ohne expliziten Kostenlink muss separat fachlich abgeglichen
werden; Repair überschreibt sie nicht und erzeugt keine zweite Position.
Ein erneut vorbereiteter bereits reparierter Datensatz blockiert; Replay der
bereits bestätigten Anfrage erzeugt keine weiteren Datensätze.
Frühere MCP-Schlüssel können alte gespeicherte Antworten wiedergeben. Für Repair
immer einen neuen Schlüssel verwenden.

## Migration und Einführung

Neue, inhaltlich identische Schema-Migrationen:

- RentalCore: `migrations/051_rental_position_cost_link.sql`
- Cores: `migrations/postgresql/049_rental_position_cost_link.sql`

Die Migration ergänzt nullable Link/Snapshot, eindeutigen Index, Linkvalidierung
und Kosten-Trigger mit Schutz gegen doppelte Skalierung durch alte Handler. Sie ist wiederholbar und verändert keine bestehenden
Zuweisungen oder Auftragspositionen. Bestehende Migrationen bleiben unverändert.
Keine neue Abhängigkeit, keine Versionspins oder Submodul-Zeiger geändert.

Schema vor dem neuen RentalCore-Code bereitstellen, danach RentalCore und MCP
koordiniert veröffentlichen und Tool-Definitionen im Client aktualisieren.
Die Cores-Init-Migration greift nur bei leerem Volume. Auf einer laufenden
Produktionsdatenbank muss ein Mensch die freigegebene Migration wie üblich von
`debian01` per SSH einspielen. Diese Aufgabe führt weder Migration noch Repair
oder Deployment produktiv aus.

## Regressionstests

- `node --test web/tests/rental-position.test.mjs` prüft den UI-Anlagepayload.
- `go test -count=1 ./...` im RentalCore prüft Preise, Netto/Brutto, Marge,
  Nullumsatz, atomare Anlage/Rollback, native UI-Reads/-Änderungen, Kosten-Snapshots,
  Tagesmodus, Replay, Guards und Repair. PostgreSQL-Tests brauchen die ausschließlich
  lokale, wegwerfbare `_test`-Datenbank über `RENTALCORE_TEST_POSTGRES_DSN`.
- MCP: `go test -count=1 ./...`, `go vet ./...`, `go build ./cmd/server`.
  `CORES_MCP_TEST_DATABASE_URL` aktiviert PostgreSQL-Referenz-/Job-Read-Tests.

Der Kosten-Normalisierungstrigger berechnet vorhandene Snapshots bei jedem
Kostenbeleg-Schreibzugriff kanonisch. Damit ist auch die zusätzliche Skalierung
älterer RentalCore-Handler während Schema-vor-Code-Rollout und Code-Rollback
unwirksam. Ein widersprüchliches gespeichertes Kosten-/Snapshot-Paar blockiert
Repair ausdrücklich, statt beim Verknüpfen still Kosten zu verändern.
