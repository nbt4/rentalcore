# RentalCore

## Auftragspositionen — Rental 5.3.121 / Warehouse 5.9.113 / MCP 1.5.59

`rental.job_positions.prepare_create/create`, `prepare_update/update` und
`prepare_archive/archive` pflegen echte Produkt-Auftragspositionen wie **+ Produkt**.
`get`, `search` und `audit_history` ergänzen gezielte Abfragen; insgesamt **442
Werkzeuge** (112 Abfragen / 165 Vorschauen / 165 Ausführungen).

Anlage verlangt Produkt/Job sowie ausdrücklich Menge und Einzelpreis (auch 0).
Produktmengen sind positive ganze Zahlen; Preise, Faktoren, Steuer und Rabatte
haben höchstens zwei Nachkommastellen. Standardwerte: Einheit Stück,
Folgetag-Faktor 0,50, Steuer 19 %, Rabatte 0; Beschreibung aus dem Produktnamen.
Teilupdates erhalten ausgelassene Felder; Job, Produkt und Herkunft bleiben
unveränderlich. Dienstleistungen, Fremdmiet- und Paketpositionen behalten ihre
nativen Abläufe; diese Tools verwalten Produktpositionen.

Die finale Vorschau zeigt Position, Preisberechnung nach den bestehenden
Job-/Steuer-/Tages-/Rabattregeln, Auftragswert und alle Materialquellen. Bereits
vorhandene manuelle Zusatzmengen benötigen ausdrücklich `manual_quantity`
(Ersatzmenge) oder `preserve_manual_quantity=true` (bewusst zusätzlicher Bedarf).
Beispiel: 9 manuelle Bedarfe mit `quantity=9, manual_quantity=0` übernehmen ergibt
9 aus Positionen + 0 manuell, nicht 18. Die Bedarfs-ID bleibt erhalten. Bereits
vorhandene Produktpositionen verlangen bei Anlage zusätzlich `allow_duplicate=true`.
Archivierung erhält Originalfelder/Historie, entfernt nur den Positionsanteil aus
aktivem Bedarf und Umsatz und lässt unabhängige Zusatzmengen bestehen.

Alle Vorschauen/Ausführungen benötigen aktuelle Rental-Adminrechte, passenden
`cores:rental:create|update|archive`-Scope (oder Legacy-write) **und separat**
`cores:rental:financial`. `get/search` benötigen den Finanzscope; die redigierte
Historie verlangt Adminrechte und enthält keine Preise. Bestätigte Ausführungen
benötigen `expected_updated_at` bei bestehenden Positionen,
`expected_job_updated_at`, `expected_context`, die genaue
`required_confirmation_text` als `confirmation_text`, `confirm_change=true`
und `idempotency_key`. Dry-run/Vorschau schreiben nichts. Veraltete Vorschauen,
fremde aktive Bearbeiter oder widersprüchliche Gerätezuordnungen sperren.
Position, genau zugehöriger Materialbedarf, Auftragswert, Job-Historie, Audit und
dauerhafte Wiederholungsantwort sind eine Transaktion. Wiederholungen prüfen
aktuelle Zielrechte erneut. Beschaffung und physischer Bestand werden nicht geändert.

Rental-Startup `050` / Suite-Migration `047` schützt Originalidentität und Archive,
versioniert alle Positionsschreiber und verhindert Gerätezuordnung zu Archiven.
Native Entfernung archiviert; native Listen und Analytics schließen Archive aus.
Warehouse-Upgrade `063` / Suite `048` aktualisiert den Produktbeziehungskontext.
Warehouse-Picklisten, Scan-Auswahl und Packlisten berücksichtigen ebenfalls nur
aktive Positionen. Rental zuerst ausrollen, danach Warehouse und MCP; keine neuen
Umgebungsvariablen. MCP-Tool-Liste neu laden, zusätzliche Finanz-/Schreibrechte bei
Bedarf über erneute OAuth-Verbindung freigeben. Bestehende Jobs/Bedarfe werden
beim Release nicht automatisch übernommen.

Validierung: vollständige Go-Tests aller drei Dienste, PostgreSQL-Tests für
atomare Übernahme, Preis-/Rabattberechnung, Zusatzmengen, Archive, veraltete
Kontexte, Rechteentzug, Rollback und dauerhafte Wiederholungen sowie Packlisten.

## Materialanforderungen — Rental 5.3.120 / Warehouse 5.9.106 / MCP 1.5.39

Existing atomic job receipts remain replayable after the requirement schema upgrade. Previously prepared unversioned requirement writes must be prepared again with the new record, parent-job and context versions before confirmation.

`rental.requirements.prepare_create/create` und `prepare_update/update` nutzen
nun eine geschlossene Rental-Owner-API mit vollständigem Mengenvorschlag.
Neu sind `get`, `search`, `audit_history` und `prepare_archive/archive`,
`prepare_restore/restore`: 353 Werkzeuge (99 Abfragen / 127 Vorschauen /
127 Ausführungen).

`quantity` ist die Gesamtmenge, `manual_quantity` die zusätzlich geplante
manuelle Menge. Positionsanteile werden aus den bestehenden Produktpositionen
berechnet und bleiben servergesteuert. Entweder Gesamt- oder manuelle Menge
angeben; bei beiden müssen die Werte übereinstimmen. Eine manuelle Menge 0
ist erlaubt, wenn eine positive Positionsmenge bestehen bleibt. Job-/Produkt-
Identität ist unveränderlich; andere Identitäten erhalten separate geprüfte Zeilen.
Exakte Zeilen-, Job- und Kontextversion, aktuelle Admin-/Aktionsrechte sowie
Vorschau und deren Bestätigungsphrase sind Pflicht. Aktive Bearbeiter und
Änderungen an Positionen, Geräten oder Produktreferenzen stoppen die Ausführung.
Native Job-Historie, Audit und dauerhafter Beleg werden atomar gespeichert;
Wiederholungen prüfen aktuelle Rechte im Zielservice.

Archive erhalten die ursprünglichen Mengen und die Zeilen-ID. Positionen und
noch zugeordnete Geräte blockieren sie. Archivierte Anforderungen zählen nicht
mehr für aktive Bedarfe, Packlisten und Produkt-/Beziehungs-Abhängigkeiten.
Restore erhält alle Felder und prüft Job, aktives Produkt und Positionsquellen.
Es gibt keine Bestandsbewegungen, Preisänderungen oder externen Nachrichten.
Native manuelle/Positions-Workflows archivieren entfernte Zeilen ebenfalls;
späteres Auswählen stellt dieselbe Identität innerhalb der Geschäftstransaktion
wieder her. Packlisten berücksichtigen zusätzliche manuelle Mengen auch neben
kommerziellen Positionen und erweitern deren Zubehör mit der gesamten Menge.

Rental `049` / Root `035` ergänzt Archivzeitpunkt, exakte monotone Zeilenversionen,
Schutz aller Schreiber und einen Index für aktive Anforderungen. Root `032` und
die Warehouse-Initialisierung behandeln archivierte Materialanforderungen als
historische Referenzen. Rental zuerst ausrollen, danach Warehouse und MCP;
frische/aktualisierte Datenbank und tatsächlichen Streamable-HTTP-Verkehr prüfen.

## Vollständige Job-Workflows — Rental 5.3.119 / MCP 1.5.38

Die bestehenden `rental.jobs.prepare_create/create` und
`rental.jobs.prepare_update/update` nutzen eine gemeinsame Rental-Owner-API.
Neu sind `prepare_archive/archive`, `prepare_restore/restore` und
`audit_history`: 346 Werkzeuge (96 Abfragen / 125 Vorschauen / 125 Ausführungen).

Alle Jobfelder sind verfügbar: Titel, Kunde, Status, Kategorie, Venue, Zeitraum,
Umsatz, Rabatt, Rabattart, Tagesmultiplikator und Steuerdarstellung. Ausgelassene
Felder bleiben erhalten; `clear_fields` leert ausschließlich nullable Zeitraum-,
Kategorie- und Venue-Felder. Planung erlaubt einen fehlenden Zeitraum; bestätigte
und abgeschlossene Jobs brauchen ein gültiges Datumspaar. Jobcode, Identität,
Revision, Sync-IDs und Endumsatz bleiben servergesteuert. Vorhandene Positionen
bestimmen den Umsatz mit derselben Berechnung wie die normale Anwendung.

Aktuelle Administrator-/Aktionsrechte, exakte Job- und Kontextversion sowie die
vollständige Vorschau und deren Bestätigungsphrase sind verpflichtend. Finanzfelder
und Änderungen berechneter Summen brauchen ausdrücklich `cores:rental:financial`;
`cores:write` ersetzt diese Freigabe nicht. Wiederholungen prüfen aktuelle Rechte
im Zielservice und geben dessen gespeicherten Beleg zurück. Aktive fremde
Bearbeitungssitzungen, neue Geräte-Zeitkonflikte oder veränderte Abhängigkeiten
stoppen Änderungen. Gleichnamige Jobs brauchen eine geprüfte, explizite
Duplikatfreigabe; archivierte Treffer werden über Restore erhalten.

Archivierung erhält alle Inhalte und beendet offene Planung als Storniert.
Ausgegebene Geräte, aktive Cases und offene Warehouse-Aufgaben blockieren sie.
Restore erhält den historischen Status und alle Inhalte; ein Wiederöffnen folgt
als eigene geprüfte Statusänderung. Abgeschlossene/stornierte Jobs schicken
noch ausgegebene Geräte in den bestehenden physischen Rückgabeprozess.
Job, native Historie, Audit und dauerhafter Wiederholungsbeleg werden atomar
verbucht. Es werden keine externen Nachrichten oder Kalendereinladungen versandt.
Die Job-Detailabfrage zeigt auch Archive und exakte Versionen; die separate
Audit-Abfrage enthält weder rohe Audits noch Sync-IDs.

Rental-Migration 048 / Umbrella-Migration 034 installieren monoton steigende
Jobversionen für alle Schreiber, Schutz archivierter Inhalte einschließlich
Positionsgeräten und Paketreservierungen sowie die vorhandenen Personal-
Zuordnungstabellen auf frischen Installationen. Rental zuerst, MCP danach deployen.

## Kunden-/Venue-Feldänderung zurücknehmen — Rental 5.3.118 / MCP 1.5.37

`rental.customers` und `rental.venues` ergänzen
`prepare_revert_update` / `revert_update`. Der Katalog enthält 341 Werkzeuge
(95 Abfragen / 123 Vorschauen / 123 Ausführungen).

Zurücknehmbar ist ausschließlich die eigene letzte MCP-Feldänderung am noch
unveränderten aktiven Datensatz. Vorschauen zeigen Quellaudit, sämtliche
fachlichen Vorher-/Nachher-Felder und Diff. Ausführung benötigt den update-Scope,
aktuelle Adminrechte, `audit_id` aus `expected_audit_id`, genaue Datensatz- und
Kontextversion, die Vorschauphrase, Bestätigung und Idempotenz. Fremde, ältere,
zwischenzeitlich bearbeitete oder bereits rückgängig gemachte Änderungen bleiben
gesperrt. IDs, Lifecycle und Sync-Referenzen werden nicht zurückgeschrieben;
zusätzliche Feldänderungen im Undo-Aufruf sind nicht erlaubt.

Felder, neuer Audit mit `reverted_audit_id` und dauerhafter Beleg werden atomar
geschrieben. Historien zeigen den Auditverweis ohne private Feldinhalte.
Erfolgreiche Belege aus 5.3.117 bleiben über den Versionswechsel identisch
wiederholbar; offene Vorschauen müssen nach dem Upgrade neu erstellt werden.

Absichtlich verschiedene gleichnamige archivierte Datensätze lassen sich jeweils
nach expliziter Duplikatprüfung mit `allow_duplicate` unter ihrer ursprünglichen
ID wiederherstellen. Neuanlage aus einem exakten Archivtreffer bleibt gesperrt.

## Kunden und Venues — RentalCore 5.3.117 / MCP 1.5.36

`rental.customers` und `rental.venues` bieten `resolve`, `get`, redigierte
`audit_history` sowie Vorschau-/Ausführungspaare für `create`, `update`, `archive`
und `restore`. Der MCP-Katalog enthält 337 Werkzeuge
(95 Abfragen / 121 Vorschauen / 121 Ausführungen).

Anlage/Pflege umfasst alle fachlichen Namens-, Rollen-, Adress-, Kontakt- und
Notizfelder. Kunden benötigen eine Identität (Firma/Name oder Vor-/Nachname)
und mindestens eine Kunden-/Lieferantenrolle; Standard ist Kunde ohne
Lieferantenrolle. Kundentypen sind Unternehmen/Privat oder leer. Postleitzahlen
bleiben Strings; E-Mail-Adressen müssen gültige reine Adressen sein.
Schema/Werkzeugparameter dokumentieren alle Feldgrenzen. Teilupdates erhalten
weggelassene Felder; explizite leere optionale Strings leeren auf null.
Sync-IDs, Bankdaten, Benutzerkonten und private Mitarbeiterdaten sind keine
schreibbaren Stammdatenfelder.

Normale Abfragen/Resolver zeigen Identität, Ort, Lebenszyklus und Version ohne
Kontakte, Anschrift, Notizen oder Sync-IDs. Berechtigte Änderungsvorschauen
zeigen sämtliche fachlichen Felder, den Diff, passende Kandidaten und aktive
Jobreferenzen. Administratorrechte und der passende Rental create/update/archive-
Scope sind erforderlich. Der MCP delegiert den ausgewählten Scope in einem
kurzlebig signierten Suite-Token; Rental prüft diesen und die tatsächlichen
aktuellen Adminrechte auch vor einem historischen Replay erneut.

Ausführung benötigt `confirm_change`, Idempotenz, genaue `expected_updated_at`
bei bestehenden Datensätzen, vollständigen SHA-256-`expected_context` und die
record-/draftgebundene Vorschauphrase. Gleiche aktive Identitäten verlangen
explizite Prüfung und `allow_duplicate`; exakte archivierte Kandidaten müssen
wiederhergestellt werden. Keine automatische Neuanlage aus Archivtreffern.

Archivierung erhält sämtliche fachlichen Felder, IDs, Sync-Referenzen und
Historie; aktive Jobs blockieren sie. Wiederherstellung erhält alle Felder,
prüft Identität/Rollen/Duplikate erneut und ist von Metadatenpflege getrennt.
Rental `047` / Root `033` schützt auch bestehende Core-Schreiber: monotone
Mikrosekundenversionen, unveränderliche IDs, keine Archivbearbeitung oder
kombinierte Lifecycle-/Feldänderung und keine physische Löschung. Aktive Jobs
benötigen aktive Kunden/Venues; historische Jobs behalten ihre Referenzen.
Venue-Listen und Jobvorschläge zeigen aktive Datensätze. Die Root-Migration
enthält das Venue-Grundschema auch für eine leere gemeinsame Installation.

Datensatz, vollständiger Vorher-/Nachher-Audit und dauerhafter Replay sind eine
Transaktion. Auditfehler hinterlassen keine Teiländerung; derselbe Schlüssel
kann erneut versucht werden. Ein erfolgreicher Replay bleibt auch nach Neustart
identisch. Historien zeigen ausschließlich Aktion, Akteur, Zeitpunkt, Version
und Lebenszyklusänderung. Die MCP-Aktion versendet keine externen Nachrichten.
Issues #4/#5 bleiben bis zum Abschluss der übrigen Completion-Bereiche offen.

## Aktive Produktbeziehungen (5.3.116)

Zubehör-/Verbrauchsmaterialvorschläge aus der gemeinsamen Warehouse-Tabelle
schließen archivierte Beziehungen sowie archivierte Quell-/Zielprodukte aus.
Archivierung erhält IDs, Felder und Historie; nach geprüfter Wiederherstellung
werden passende Beziehungen erneut angeboten. WarehouseCore 5.9.105 / Umbrella
032 liefert das gemeinsame Schema und die vollständigen MCP-Lifecycle-Werkzeuge.
Ein PostgreSQL-Integrationstest prüft Filter, Restore und unveränderte Historie;
Frontend-Build und vollständige Go-/Race-Tests gehören zur Release-Prüfung.


## Analyse und Jobpositionen (5.3.115)

Die Umsatzanalyse zeigt standardmäßig nur abgeschlossene Jobs als realisierten
Umsatz. Über „Umsatzansicht“ lässt sich separat die Pipeline aus geplanten und
bestätigten Jobs anzeigen; stornierte und archivierte Jobs zählen in keiner der
beiden Ansichten. Die Zeiträume beziehen sich für realisierten Umsatz auf das
Enddatum und für die Pipeline auf das Startdatum. Der Monatsverlauf folgt
derselben Zuordnung. Im Drilldown lassen sich auch einzelne Dienstleistungen,
Produkte und Geräte öffnen; darunter stehen die zugehörigen Jobs mit
Jobnummer, Titel, Position und einem Link zum Jobdetail.

Im Jobdetail bezeichnet „Auftragspositionen · Produkte“ die kaufmännischen
Positionen, aus denen Auftragswert und automatischer Produktbedarf entstehen.
Ein Link führt von dort direkt zu „Material und Geräte“ für die Zuordnung der
konkreten Geräte. Diese Unterscheidung ist auch im englischen UI erklärt.

`GET /api/v1/analytics/revenue/drilldown` verwendet standardmäßig
`scope=realized`; für geplante und bestätigte Jobs gilt `scope=pipeline`.
`period=30days|90days|1year|all` filtert je nach Ansicht zurück- oder
vorausblickend. Blattknoten enthalten zusätzlich `jobs` mit Job- und
Positionsbezug.

## Job-Arbeitsbereich und Joblogik (5.3.114)

Die Jobübersicht bündelt Suche, Statusfilter und Kennzahlen. Neue Jobs beginnen in
`Planung` mit Kunde und Titel; Start- und Enddatum müssen gemeinsam angegeben
werden. Zur Bestätigung ist ein Zeitraum erforderlich. Statuswechsel folgen
`Planung → Bestätigt → Abgeschlossen`; Stornieren ist aus offenen Zuständen
möglich. Abgeschlossene oder stornierte Jobs können zur weiteren Bearbeitung
zunächst wieder in `Planung` gesetzt werden.

Im Jobdetail werden Stammdaten, Positionen, zusätzlich geplantes Material,
zugewiesene Geräte, Personal, Dokumente und Verlauf verwaltet. Produktpositionen
erzeugen automatisch Materialbedarf. Im Formular gewählte Produkte zählen als
zusätzlicher manueller Bedarf und bleiben beim Ändern von Positionen erhalten.
Positionspreise, Veranstaltungstage, Rabatt und Steuer werden serverseitig
berechnet; das Zuweisen eines Geräts überschreibt den Umsatz bei Jobs mit
Positionen nicht. Der API-Update-Body kann `revision` enthalten; bei einer
veralteten Revision antwortet RentalCore mit `409`, damit Änderungen anderer
Bearbeiter nicht überschrieben werden. Die Oberfläche sendet diese Revision.
Detail und Formular wechseln auf eine Spalte, sobald die Navigation den
Arbeitsbereich verengt. Positionsfelder brechen innerhalb ihrer Karte um;
lange Auftragsdaten bleiben in der rechten Spalte lesbar.

Eine erneute PDF-Finalisierung ersetzt nur Positionen, die dem Import
zugeordnet sind. Ältere Positionen ohne Herkunftsmarkierung bleiben bestehen;
identische alte Zeilen werden beim erneuten Import nicht verdoppelt. Die
Migration ergänzt `jobs.revision`, `job_positions.pdf_extraction_item_id` und
die getrennten Mengen `manual_quantity`/`position_quantity` beim Start. Der
Wert `quantity` bleibt deren Summe für WarehouseCore.

`DELETE /api/v1/jobs/:id` archiviert einen Job mit `deleted_at`; Historie,
Positionen und Gerätebeziehungen bleiben erhalten. Ausgegebene Geräte müssen
vorher zurückgenommen werden, sonst antwortet die API mit `409`. Das Archivieren
entfernt den Kalendereintrag auch nach dem Soft Delete. Beim Start bereinigt
RentalCore außerdem noch vorhandene Termine bereits archivierter Jobs und
wiederholt damit fehlgeschlagene Löschungen. Health-Endpunkt und strukturierte Logs
melden die veröffentlichte RentalCore-Version. Dokumente werden über den File Pool gespeichert;
ohne Nextcloud bleiben die Dateien im persistenten Volume `/app/uploads`.

## Gemeinsames Etikettenbogen-Vokabular (5.3.111)

Die suiteweiten Deutsch-/Englisch-Ressourcen enthalten jetzt auch A4-
Etikettenbögen, Papieroptionen, individuelle Stückzahlen und dynamische
Druckmeldungen des WarehouseCore-Druckcenters.

## Gemeinsames Datentransfer-Vokabular (5.3.110)

Die synchronisierten Deutsch-/Englisch-Ressourcen enthalten jetzt auch die
suiteweit verwendeten Datensatz-, Feld-, Vorschau- und Konfliktbegriffe des
zentralen Import-/Export-Arbeitsbereichs. Fachliche Nutzdaten und technische
Spaltenschlüssel bleiben dabei unverändert.

## Vollständige Dashboard-Lokalisierung (5.3.109)

Deutsch und Englisch funktionieren nun auch bei gemischten Quelltexten
bidirektional. Das Rental-Dashboard übersetzt Kennzahlen, Terminradar,
Schnellaktionen sowie dynamische Anzahlen und Fristen vollständig in die
gewählte Suite-Sprache.

## Jev-gestütztes OCR-Matching (5.3.108)

Nach gespeicherten und exakten Zuordnungen entscheidet Jev über OpenRouter
zwischen begrenzten Katalogkandidaten für Produkte, Pakete, Mietmaterial und
Dienstleistungen. Die bestehende OCR-Extraktion, Preise, Mengen, Kundenzuordnung
und Jobanlage bleiben lokal und deterministisch. Ohne `OPENROUTER_API_KEY`, bei
Timeout oder unterhalb `JEV_MIN_CONFIDENCE` greift das bisherige Matching. An
OpenRouter gehen nur eine einzelne Positionsbeschreibung und die zugehörigen
Kandidatenstammdaten, nie das gesamte Dokument oder Kundendaten.

## Geführte Änderungen an Produktbedarfen (5.3.107)

`PUT /api/v1/jobs/:id/requirements/:requirementId` ändert ausschließlich die
positive Menge einer vorhandenen Produktbedarfszeile. Job- und Produktbezug
bleiben unveränderlich. Alte und neue Menge werden mit Benutzerkontext in der
Job-Historie protokolliert, sodass die Änderung geprüft und manuell
zurückgenommen werden kann.

## Release 5.3.106 – Korrekte Versionsmeldung

Der Health-Endpunkt meldet die aktuelle Release-Version, damit Dashboard und
Betriebsüberwachung den ausgerollten Stand eindeutig anzeigen.

## Deutsch und Englisch (5.3.105)

Die Sidebar bietet die gemeinsame Cores-Sprachwahl. Navigation, Job- und
Kundenbegriffe, zentrale Aktionen, zugängliche Beschriftungen, Datumsformat und
Begrüßung folgen der suiteweit gespeicherten Auswahl.

## Additive Produktbedarfe für Integrationen (5.3.105)

`POST /api/v1/jobs/:id/requirements` legt genau einen neuen Produktbedarf für
einen bestehenden Job an. Job, aktives Produkt und positive Menge werden
serverseitig geprüft; eine vorhandene Job-Produkt-Verknüpfung liefert `409` und
wird niemals überschrieben. Der Vorgang erscheint außerdem in der Job-Historie.

## Suite-Navigation (5.3.104)

Am Ende der Fachnavigation steht jetzt dieselbe Core-Auswahl wie in allen
anderen Cores. Der eigenständige Dashboard-Link bleibt direkt darunter
erreichbar; beide Ziele funktionieren auf separaten Domains und im gemeinsamen
Pfadmodus.

## Einheitliches Cores Designsystem

RentalCore folgt im React-Client und in den verbliebenen Go-Templates dem verbindlichen Designvertrag aus [`nbt4/cores`](https://github.com/nbt4/cores/blob/main/docs/DESIGN_SYSTEM.md). Palette, Inter-Typografie, Größenleiter, 256/80-px-Sidebar, Tabellen, Eingaben, Selects, Dropdowns, Scrollbars und Dashboard-Hierarchie sind mit allen anderen Cores identisch.

`web/src/cores-theme.css`, `web/src/lib/cores-design.ts` und `web/static/css/cores-theme.css` sind generiert und dürfen nicht direkt geändert werden. Änderungen erfolgen in `cores/theme/` und werden mit den Umbrella-Skripten synchronisiert und geprüft.

**Kernservice für Vermietungsmanagement im Cores-Ökosystem — Auftragsverwaltung, Kundendaten, Gerätezuweisung, Barcode-Generierung und automatisierte OCR-Belegverarbeitung.**

---

## Features

- **Auftragsmanagement (Jobs)** — Vollständiger CRUD-Workflow mit dem verbindlichen Lebenszyklus `Planung → Bestätigt → Abgeschlossen` sowie `Storniert` als Abbruchstatus
- **Operatives Dashboard** — Personalisierte Tagesübersicht mit aktiven, laufenden, anstehenden und überfälligen Jobs, monatlichem Auftragswert, Terminradar und direkten Arbeitswegen
- **Kundenverwaltung** — CRM mit Kontaktdaten, Historie und verknüpften Aufträgen; bei deutschen fünfstelligen PLZ wird der Ort automatisch vorgeschlagen und bleibt manuell änderbar
- **Gerätezuweisung** — Zuweisung und Entfernung von Devices zu/von Aufträgen. Verfügbarkeitsprüfung in Echtzeit
- **Kontextuelle Produktsuche** — Produkt-, Geräte-, Paket-, Mietprodukt-, Dienstleistungs- und PDF-Zuordnungssuchen berücksichtigen Marke, Hersteller, Kategorien, Identifikatoren und technische Stammdaten; kombinierte Begriffe dürfen über mehrere Felder verteilt sein
- **Barcode- und QR-Generierung** — Automatische Erstellung von QR-Codes und Barcode-Labels (Barcode128) pro Gerät/Seriennummer
- **OCR-Belegverarbeitung** — Python-3.12-Pipeline zur Extraktion von Dokument-/Jobtiteln, mehrzeiligen Positionsbeschreibungen, Mengen, Preisen und Positionsrabatten; erkannte Jobtitel sind vor dem Finalisieren editierbar, neue Produkt-Katalogentwürfe können sicher und duplikatgeprüft angelegt werden, während Klassifizierung und physische Geräte bewusst in WarehouseCore gepflegt werden
- **Jev-Entscheidungsabgleich** — Optionales semantisches OCR-Matching gegen Produkte, Pakete, Mietmaterial und Dienstleistungen mit Confidence-Schwelle und lokalem Fallback
- **M365-Kontaktsync** — Bidirektionale Synchronisation mit Microsoft 365 Shared-Mailbox-Kontakten über die zentrale Cores-App-Registrierung
- **Nextcloud Filepool** — WebDAV-basierte Dateiablage für auftragsbezogene Dokumente mit automatischer Zuweisung
- **Passkey / WebAuthn** — Passwortlose Authentifizierung mit FIDO2/WebAuthn (Passkeys)
- **Analytics Dashboard** — Realisierter Umsatz abgeschlossener Jobs und separate Pipeline geplanter oder bestätigter Jobs; interaktiver Netto-/Brutto-Umsatz-Drilldown aus den live synchronisierten Auftragspositionen über eigene Produkte, Mietprodukte samt Lieferantenkosten/Marge und Dienstleistungen bis zum konkreten Job und Einzelgerät; Mietkosten folgen dabei der Auftragseinstellung „Preis × Veranstaltungstage“
- **Installierbare Mobile-App (PWA)** — Standalone-Modus mit RentalCore-App-Icon, Safe-Area-Unterstützung, großen Touch-Zielen, App-Tabbar und Drawer-Navigation; eigenständig installierbar und zusätzlich unter `/rentalcore/` nahtlos innerhalb der installierten Cores-PWA nutzbar
- **Zentrales Branding** — Semantische RentalCore-Logos in Sidebar, Login, Favicon und PWA; Rechnungen, HTML-E-Mails und Geräteetiketten verwenden getrennt davon die zentrale Unternehmensmarke
- **Einheitliche Navigation** — Ein-/ausklappbare Sidebar mit suite-weitem Core-Auswahlfeld und eigenständigem Dashboard-Link an derselben Position in allen Cores

---

## Job-Lebenszyklus

RentalCore speichert genau vier Job-Stati: `Planung`, `Bestätigt`, `Abgeschlossen` und `Storniert`. Nur bestätigte Jobs sind im WarehouseCore zur Vorbereitung und Ausgabe freigegeben. `Aktiv` wird aus dem bestätigten Status und dem Veranstaltungszeitraum abgeleitet; Packfortschritt, Geräterücklauf und Rechnungszustand bleiben eigenständige Dimensionen. Insbesondere ist `Abgerechnet` kein Jobstatus: Ein beendeter Job bleibt `Abgeschlossen`, während Rechnungen separat beispielsweise `draft`, `sent` oder `paid` durchlaufen.

Beim Wechsel auf `Abgeschlossen` oder `Storniert` werden bereits ausgegebene Geräte auf `return_pending` gesetzt. Erst die physische Rücknahme mit Lagerplatzbestätigung setzt sie wieder auf `in_storage`; der Job bleibt dabei `Abgeschlossen` beziehungsweise `Storniert`.

Die Startmigration konsolidiert historische deutsche und englische Werte automatisch. `Vorbereitung`, `Aktiv`, `open`, `in progress` und `Pausiert` werden zu `Bestätigt`; `Abgerechnet`, `Completed` und `paid` zu `Abgeschlossen`; Stornovarianten zu `Storniert`.

---

## Tech-Stack

| Schicht       | Technologie                                        |
|---------------|----------------------------------------------------|
| Backend       | Go 1.24, Gin, GORM, PostgreSQL 16                  |
| Frontend      | React 19, TypeScript, Vite 7, Zustand 5            |
| Styling       | Tailwind CSS 4, PostCSS, Tailwind Merge            |
| Auth          | JWT (golang-jwt/jwt/v5), bcrypt, WebAuthn          |
| Barcodes      | boombuler/barcode, skip2/go-qrcode                 |
| PDF           | jung-kurt/gofpdf                                   |
| OCR           | Python 3 venv mit custom OCR-Pipeline              |
| Dateiablage   | Nextcloud WebDAV                                   |
| Container     | Docker (Multi-Stage: Node 20 + Go 1.25 + Python 3.12 Alpine) |

---

## Schnellstart

### Docker

```bash
docker run -d \
  --name rentalcore \
  -e DB_HOST=postgres \
  -e DB_PORT=3306 \
  -e DB_NAME=rentalcore \
  -e DB_USERNAME=rentalcore_user \
  -e DB_PASSWORD=yourpassword \
  -e ENCRYPTION_KEY=your-256-bit-encryption-key \
  -e GIN_MODE=release \
  -p 8080:8080 \
  nobentie/rentalcore:latest
```

### docker-compose (Auszug)

```yaml
rentalcore:
  image: nobentie/rentalcore:latest
  ports:
    - "8081:8080"
  environment:
    DB_HOST: postgres
    DB_PORT: 5432
    DB_NAME: rentalcore
    DB_USER: rentalcore
    DB_PASS: ${DB_PASS}
    ENCRYPTION_KEY: ${ENCRYPTION_KEY}
    GIN_MODE: release
    CORES_JWT_SECRET: ${CORES_JWT_SECRET}
    NEXTCLOUD_WEBDAV_URL: ${NEXTCLOUD_WEBDAV_URL}
  depends_on:
    - postgres
  volumes:
    - rental_uploads:/app/uploads
```

---

## API-Endpunkte

Die React-Oberfläche besitzt keinen eigenen interaktiven Login mehr. Bei einer
fehlenden Sitzung öffnet sie den zentralen Cores-Login und übergibt die aktuelle
RentalCore-Ansicht als Rücksprungziel; damit steht dort auch Microsoft Entra zur
Verfügung. Die Auth-Endpunkte bleiben für kompatible API-Clients bestehen.

| Methode  | Pfad                                        | Beschreibung                                  |
|----------|---------------------------------------------|-----------------------------------------------|
| `POST`   | `/api/v1/auth/login`                        | Benutzer-Login                                |
| `POST`   | `/api/v1/auth/logout`                       | Session beenden                               |
| `GET`    | `/api/v1/auth/me`                           | Aktuellen Benutzer abrufen (🔒)                |
| `POST`   | `/api/v1/auth/change-password`              | Passwort ändern (🔒)                           |
| `GET`    | `/jobs`                                     | Auftragsliste (🔒)                             |
| `POST`   | `/jobs`                                     | Neuen Auftrag erstellen (🔒)                   |
| `GET`    | `/jobs/:id`                                 | Auftragsdetails (🔒)                           |
| `PUT`    | `/jobs/:id`                                 | Auftrag aktualisieren (🔒)                     |
| `DELETE` | `/jobs/:id`                                 | Auftrag löschen (🔒)                           |
| `GET`    | `/jobs/:id/devices`                         | Zugewiesene Geräte abrufen (🔒)                |
| `POST`   | `/jobs/:id/devices`                         | Gerät zuweisen (🔒)                            |
| `DELETE` | `/jobs/:id/devices/:deviceId`               | Gerät entfernen (🔒)                           |
| `GET`    | `/customers`                                | Kundenliste (🔒)                               |
| `GET`    | `/api/v1/customers/postal-code/:postalCode` | Orte zu deutscher PLZ auflösen (🔒)            |
| `POST`   | `/customers`                                | Neuen Kunden anlegen (🔒)                      |
| `PUT`    | `/customers/:id`                            | Kunden aktualisieren (🔒)                      |
| `DELETE` | `/customers/:id`                            | Kunden löschen (🔒)                            |
| `GET`    | `/devices/:id`                              | Gerätedetails (🔒)                             |
| `GET`    | `/devices/:id/stats`                        | Gerätestatistiken (🔒)                         |
| `GET`    | `/devices/available`                        | Verfügbare Geräte (🔒)                         |
| `GET`    | `/barcodes/device/:serialNo/qr`             | QR-Code pro Gerät generieren (🔒)              |
| `GET`    | `/barcodes/device/:serialNo/barcode`        | Barcode pro Gerät generieren (🔒)              |
| `GET`    | `/analytics/revenue`                        | Umsatz-Analytics (🔒)                          |
| `GET`    | `/analytics/revenue/drilldown`              | Hierarchischer Umsatz-Drilldown (🔒)           |
| `GET`    | `/analytics/equipment`                      | Equipment-Analytics (🔒)                       |
| `GET`    | `/statuses`                                 | Status-Liste (🔒)                              |
| `GET`    | `/health`                                   | Health Check (öffentlich)                      |

🔒 = Authentifizierung via `session_id` Cookie erforderlich

---

## Umgebungsvariablen

| Variable                       | Beschreibung                                      | Standard               |
|--------------------------------|---------------------------------------------------|------------------------|
| `DB_HOST`                      | Datenbank-Host                                    | –                      |
| `DB_PORT`                      | Datenbank-Port                                    | `3306`                 |
| `DB_NAME`                      | Datenbank-Name                                    | `rentalcore`           |
| `DB_USERNAME`                  | Datenbank-Benutzer                                | –                      |
| `DB_PASSWORD`                  | Datenbank-Passwort                                | –                      |
| `ENCRYPTION_KEY`               | 256-Bit-Verschlüsselungs-Key                      | –                      |
| `SESSION_TIMEOUT`              | Session-Timeout in Sekunden                       | `3600`                 |
| `GIN_MODE`                     | Gin-Modus (`release` oder `debug`)                | `release`              |
| `POSTAL_LOOKUP_BASE_URL`       | Basis-URL der serverseitigen deutschen PLZ-Suche  | `https://openplzapi.org/de` |
| `NEXTCLOUD_WEBDAV_URL`         | Nextcloud WebDAV-URL für Filepool                 | –                      |
| `NEXTCLOUD_WEBDAV_USER`        | Nextcloud WebDAV-Benutzer                         | –                      |
| `NEXTCLOUD_WEBDAV_PASSWORD`    | Nextcloud WebDAV-Passwort                         | –                      |
| `NEXTCLOUD_WEBDAV_BASE_PATH`   | WebDAV-Basispfad                                  | `rentalcore-filepool`  |
| `FILEPOOL_ASSIGNED_ROOT`       | Root-Pfad für zugewiesene Dateien                 | `assigned`             |
| `FILEPOOL_UNASSIGNED_ROOT`     | Root-Pfad für unzugewiesene Dateien               | `unassigned`           |
| `SMTP_HOST`                    | SMTP-Host für E-Mail-Versand                      | –                      |
| `SMTP_PORT`                    | SMTP-Port                                         | `587`                  |
| `SMTP_USERNAME`                | SMTP-Benutzername                                 | –                      |
| `SMTP_PASSWORD`                | SMTP-Passwort                                     | –                      |
| `M365_TENANT_ID`               | Entra ID Tenant-ID (für Kontaktsync)              | –                      |
| `M365_CLIENT_ID`               | Entra ID Client-ID                                | –                      |
| `M365_CLIENT_SECRET`           | Entra ID Client-Secret                            | –                      |
| `M365_SHARED_MAILBOX_ID`       | Shared Mailbox-ID                                 | –                      |
| `M365_SYNC_INTERVAL`           | Sync-Intervall (z. B. `5m`)                       | `5m`                   |
| `M365_CALENDAR_MAILBOX`        | Raum-Mailbox für den zentralen Jobkalender        | `events-calender@tsunami-events.de` |
| `WAREHOUSECORE_DOMAIN`         | WarehouseCore-Domain für Cross-Navigation         | –                      |
| `CORES_JWT_SECRET`             | JWT-Secret (Cores-weit identisch)                 | –                      |
| `OPENROUTER_API_KEY`           | Aktiviert optional Jev über OpenRouter            | –                      |
| `JEV_ENABLED`                  | Jev explizit aktivieren/deaktivieren              | `true`                 |
| `JEV_API_URL`                  | OpenRouter Decisions-Endpunkt                     | `https://openrouter.ai/api/alpha/decisions` |
| `JEV_MODEL`                    | Jev-Modell-ID                                     | `typesafe/jev-1.13`    |
| `JEV_TIMEOUT`                  | Maximale Dauer je Decision-Request                | `3s`                   |
| `JEV_MIN_CONFIDENCE`           | Mindestkonfidenz für eine Jev-Zuordnung            | `0.70`                 |

Die `M365_*`-Variablen bleiben als Fallback bestehen. Sobald im Cores-Dashboard unter **Microsoft 365 & Entra** Werte gespeichert sind, lädt RentalCore Tenant-ID, Client-ID, Secret, Mailboxen, Intervall und App-URL beim Start aus der gemeinsamen Tabelle `m365_settings`. Damit wird für Entra-Benutzer, Login, Kontakte und Kalender nur eine registrierte Tenant-App benötigt.

Die Ortssuche läuft serverseitig über OpenPLZ und speichert Ergebnisse für 24 Stunden im Arbeitsspeicher. An den Dienst wird ausschließlich die eingegebene fünfstellige PLZ übertragen. Ist die Suche nicht erreichbar oder liefert sie keinen Treffer, bleibt das Ortsfeld frei editierbar.

Für den Jobkalender muss `M365_CALENDAR_MAILBOX` eine Exchange-Online-Raumressource sein. RentalCore legt dort genau einen Termin je Job an und synchronisiert alle zugewiesenen Bearbeiter als erforderliche Teilnehmer desselben Meetings. Antworten und neue Zeitvorschläge sind deaktiviert; RentalCore bestätigt die Teilnehmerinstanz über Microsoft Graph ohne Antwortmail. So erscheint kein zusätzlich von RentalCore erzeugter Einzeltermin im Bearbeiterkalender. Eine bestehende Mailbox kann in Exchange Online PowerShell vorbereitet werden:

```powershell
Set-Mailbox events@example.com -Type Room
Set-CalendarProcessing events@example.com -AutomateProcessing AutoAccept -AllowConflicts $true -AllBookInPolicy $true -DeleteSubject $false -DeleteComments $false -AddOrganizerToSubject $false
```

Die App-Registrierung benötigt dafür die Microsoft-Graph-Anwendungsberechtigung `Calendars.ReadWrite` mit Administratorzustimmung. Kontakt- und Kalender-Sync können unabhängig voneinander verwendet werden; für reinen Kalendersync ist `M365_SHARED_MAILBOX_ID` nicht erforderlich.

---

[Quellcode](https://github.com/nbt4/rentalcore) | [Monorepo](https://github.com/nbt4/cores) | `nobentie/rentalcore:latest`
# Release 5.3.103

Das Kundenformular ergänzt zu einer deutschen fünfstelligen PLZ automatisch den Ort;
bei mehreren Treffern kann der passende Ort ausgewählt und jederzeit manuell geändert
werden. Die OCR-Übernahme berechnet und speichert Positionsrabatte aus Einzel- und
Zeilengesamtpreis. Nach dem Finalisieren einer OCR-Erkennung wird der Job unmittelbar
mit dem zentralen M365-Raumkalender synchronisiert. Die Startreparatur ergänzt bereits
verpasste zukünftige Jobtermine weiterhin automatisch.

# Release 5.3.102

Die OCR-Pipeline übernimmt Dokumentüberschriften wie `Angebot Luther Theater AG0081`
als editierbaren Jobtitel und erkennt Dokumenttyp sowie Dokumentnummer. Mehrzeilige
Positionsbeschreibungen bleiben auch dann eine Position, wenn eine Detailzeile mit
einer Zahl beginnt; durch PDF-Zeilenumbrüche getrennte Wörter werden wieder verbunden.
Seitenfuß-, Steuer- und Summenwerte fließen nicht mehr als vermeintliche Preise in
Positionen ein, und deutsche Tausenderbeträge werden korrekt gelesen.

# Release 5.3.101

Der Microsoft-Kalendersync verwendet eine Exchange-Raumressource als zentralen
Jobkalender. Jeder Job besitzt genau einen Raumtermin; Bearbeiter werden als
Teilnehmer desselben Meetings automatisch und ohne Antwortmail bestätigt.
Bestehende eigenständige Bearbeitertermine werden beim nächsten Job-Sync bereinigt.
Kontakt- und Kalendersync lassen sich unabhängig voneinander konfigurieren.
Beim Start gleicht RentalCore ausschließlich zukünftige Jobs mit fehlendem Raumtermin
oder vorhandenen Alttermin-IDs ab; erfolgreich migrierte Jobs werden bei späteren
Neustarts nicht erneut aktualisiert.
