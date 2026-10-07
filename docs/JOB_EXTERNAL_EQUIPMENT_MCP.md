# Mietprodukte am Job über MCP zuweisen

`rental.job_external_equipment.prepare_create` und
`rental.job_external_equipment.create` weisen ein **vorhandenes aktives**
Fremdmietprodukt einem bestehenden Job unter „Mietprodukte“ zu. Der MCP-Katalog
enthält damit 444 Werkzeuge. Es wird kein neuer Katalogartikel angelegt.

1. Mit `rental.external_equipment.list` das Katalogprodukt und mit
   `rental.jobs.get` den Job finden.
2. `prepare_create` mit `job_id` oder `job_query`, `equipment_id` oder
   `equipment_query`, einer ausdrücklich angegebenen `quantity` (1–1000) und
   `days_used` (1–365) aufrufen. Mehrdeutige Namen erfordern eine Auswahl.
3. Die gesamte Vorschau mit Lieferant, Lieferanten-Tagesmietpreis (`rental_price`),
   Kundenpreis (`customer_price`), Menge, Mietdauer und Gesamtkosten zeigen.
   Gesamtkosten folgen `rental_price × quantity × days_used`; bei deaktiviertem
   `multiply_by_days` zählt ein Tag. Fehlende Lieferantenpreise blockieren die
   Anlage. Ein ausdrücklich gespeicherter Preis von 0 bleibt gültig.
4. Nach ausdrücklicher Bestätigung `create` mit den aufgelösten IDs,
   unveränderten Eingaben, `expected_job_updated_at`, `expected_context`,
   `confirmation_text` aus der finalen Vorschau, `confirm_change: true` und einem
   stabilen `idempotency_key` aufrufen. Jede weitere Zuordnung braucht eine neue
   Vorschau: Die vorherige Anlage ändert die Jobversion.

Aktuelle Administratorrechte, `cores:rental:create` (oder bestehendes
`cores:write`) und separat ausgewähltes `cores:rental:financial` sind erforderlich.
Die Werkzeuge veröffentlichen die benötigten OAuth-Scopes und liefern bei fehlender
Freigabe eine passende OAuth-Challenge. `dry_run` erzwingt eine Vorschau.

RentalCore führt den geschlossenen Vorgang über
`POST /api/v1/mcp/jobs/external-equipment-create` aus. Signierte persönliche
MCP-Delegation, Live-Rechteprüfung und exakte Vorschau sind auch dort erforderlich.
Bestehende Zuordnungen werden als Konflikt gezeigt und niemals überschrieben.
Veränderte Jobfelder, Katalogpreise, Zuordnungen oder aktive Bearbeiter sperren
veraltete Bestätigungen. Zuordnung, Jobversion, Jobhistorie, Audit und dauerhafter
Wiederholungsbeleg werden in einer Transaktion gespeichert; dieselbe bestätigte
Anfrage mit demselben Schlüssel liefert das gespeicherte Ergebnis zurück.

Der Kundenpreis wird zur Prüfung gezeigt. Die Zuordnung speichert die Mietkosten
in `job_rental_equipment`, erzeugt keine kommerzielle Auftragsposition und
berechnet den Jobumsatz nicht neu. Eigenbestands-Anforderungen, Lagerbewegungen
und Procurement bleiben unverändert. Das vorhandene Schema und die Migration
039 werden verwendet; es gibt keine neue Migration oder Abhängigkeit.

Geplanter Release: RentalCore 5.3.122 und Cores-MCP 1.5.61. RentalCore zuerst
ausrollen, danach Cores-MCP. Anschließend die MCP-Tool-Definitionen im Client
aktualisieren. Die Tools werden erst nach Veröffentlichung beider Dienständerungen
verfügbar. Es sind keine zusätzlichen produktiven Schemaänderungen vorgesehen.
