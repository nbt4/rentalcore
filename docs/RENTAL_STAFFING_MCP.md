# Personalzuordnung über MCP

RentalCore **5.3.129**, Cores MCP **1.5.68**. Issues nbt4/cores-mcp#4 und #5.
Gesamtkatalog **477 Tools: 123 lesend, 177 vorbereitend, 177 ausführend**.

## Ablauf und vollständiges Fachmodell

`rental.employees.search` löst bestehende Mitarbeiter anhand von Namen oder
Qualifikation auf. Ergebnis: ID, Name, Aktivstatus und Qualifikations-ID/-Name/
-Kategorie. `include_inactive` ist für Historie und eindeutige Identifizierung
verfügbar. `rental.skills.search` durchsucht Qualifikationsnamen/Kategorie.
Kein Mitarbeiter wird über diese Werkzeuge neu angelegt oder privat gepflegt.

Entity-Schema: `cores.entities.schema` mit `entity=rental.staffing`.
Die Zuordnung hat die unveränderliche Identität `job_id` / `employee_id`, eine
optionale `role` bis100 Unicode-Zeichen und die Termine des Jobs. Es gibt im
bestehenden Fachmodell keine separaten Schichtzeiten, Personalpreise oder
verpflichtende Qualifikationsvorgaben. Qualifikationen dienen der Auswahl.

| Vorbereitung | Ausführung | Wirkung |
| --- | --- | --- |
| `rental.staffing.prepare_create` | `rental.staffing.create` | Aktiven Mitarbeiter einem bestehenden aktiven Job zuweisen |
| `rental.staffing.prepare_update` | `rental.staffing.update` | Rolle ändern, leere Zeichenkette leert; kein impliziter Identitätswechsel |
| `rental.staffing.prepare_archive` | `rental.staffing.archive` | Zuordnung behalten, aus aktueller Personalplanung entfernen |
| `rental.staffing.prepare_restore` | `rental.staffing.restore` | Behaltene Zuordnung mit unveränderter Rolle und aktivem Mitarbeiter wiederherstellen |

`role` wird bei Create optional angegeben, bei Update explizit verlangt und
bei Archive/Restore nicht akzeptiert. Weglassen erhält den Wert, JSON-null
ist kein Clear-Befehl. Vorhandene aktive oder archivierte Identitäten blockieren
Create; Update oder Restore verwenden. `get` liest genau ein Job/Mitarbeiter-
Paar, `search` einen Job (`include_archived` für Restore) und `audit_history`
den begrenzten nativen/MCP-Änderungsverlauf mit Rolle und Ergebnisversion.

## Entwurf, Sperren und Bestätigung

Alle acht Schreib-/Vorbereitungswerkzeuge verlangen `cores:read`, einen echten
Suite-Nutzer, aktuelle Rental-Adminrechte und den ausdrücklich erteilten Scope
`cores:rental:create`, `cores:rental:update` oder `cores:rental:archive`.
Restore verwendet Archive-Rechte. `cores:write` allein und Finanzrechte erteilen
diese neue Personalaktion nicht. OAuth-Tooldeskriptor und `_meta` veröffentlichen
die zwei passenden Scopes; fehlende Zustimmung liefert eine OAuth-Nachfreigabe.
Reine Personalzugriffe brauchen keinen Finanzscope.

Vorbereitung erzeugt keinen Geschäftsdatensatz, Audit oder Kalenderauftrag.
Der finale Entwurf zeigt aktuellen Zustand, vollständige Zuordnung, Diff,
minimale Job-/Mitarbeiterkontexte, Qualifikationen, aktive fremde Bearbeiter,
Terminkonflikte und Kalenderwirkung. Ausführung übernimmt exakt
`expected_job_updated_at`, bei bestehenden Zuordnungen `expected_updated_at`,
`expected_context`, `required_confirmation_text` als `confirmation_text`,
`confirm_change=true` und einen eindeutigen `idempotency_key`.
`dry_run=true` entfernt die Bestätigung und führt nur die Vorschau aus.

Terminkonflikte vergleichen inklusive Job-Tagesgrenzen mit anderen nicht
archivierten, nicht abgeschlossenen/stornierten Jobs des Mitarbeiters. Für
Create/Restore mit Konflikten ist nach ihrer Prüfung `allow_overlap=true`
erforderlich; danach erneut final vorbereiten/bestätigen. Mehr als100 Konflikte
blockieren den Vorgang. Jobs ohne Startdatum liefern keine Verfügbarkeitszusage.
Geänderte Job-/Mitarbeiter-/Qualifikations-/Konfliktkontexte machen alte Entwürfe
ungültig. Fremde aktive Job-Bearbeitung blockiert. NOWAIT-Tabellen-/Zeilensperren
liefern bei Schreibkollision409 und vermeiden umgekehrte Child/Job-Wartezyklen.

## Atomarität, Datenschutz und native Kompatibilität

RentalCore erhält nur eine geschlossene typisierte32KiB-Anfrage und prüft die
signierte echte Identität sowie die ausgewählte Aktion vor dem Datenzugriff.
Aktuelle DB-Admin-/Aktivrechte gelten auch vor dauerhaftem Receipt-Replay.
Zuordnung, monotone Mikrosekunden-/Jobversion, nativer Jobverlauf, Audit und
Idempotenzbeleg committen zusammen oder werden vollständig zurückgerollt.
Gleicher Schlüssel/Request liefert den gespeicherten Erfolg, abweichender
Request409. Replays prüfen aktuelle Rechte und senden keinen zweiten Kalender-
auftrag. Normale Leseabfragen bleiben read-only und filtern aktuellen DB-Admin
vor Ergebnissen, Joins und Pagination.

Keine privaten Adressen, Geburtsdaten, E-Mail-/Telefonangaben, IBAN, privaten
Notizen, Kalender-IDs oder vollständigen Mitarbeiter-JSONs werden ausgegeben
oder im Personal-Audit gespeichert. Die Rollen sind nutzereingegebener Text.
Native delegierte Mitarbeiter-/Personalzugriffe sind vor DB-Zugriff gesperrt;
für MCP gelten ausschließlich die benannten geführten Werkzeuge.

Native UI-Zuweisung und Entfernung behalten dieselbe Identität, auditieren
atomar und entfernen über Soft-Archive. Erneutes natives Zuweisen stellt die
Zuordnung wieder her; eine dabei ausdrücklich gesendete neue Rolle wird nach
dem Restore in einer zweiten SQL-Änderung innerhalb derselben Transaktion
übernommen. Bestehende aktive Zuweisung bleibt beim erneuten Add unverändert.
Archivierte Zuordnungen fehlen in aktuellen Staffing-Abfragen, Jobkontexten und
Kalenderteilnehmern; historische Zuweisungen bleiben direkt lesbar.
Mitarbeiter mit behaltenen Zuordnungen können nicht mehr hart gelöscht werden:
Deaktivierung verwenden. Eine spätere Jobwiederherstellung prüft nur die aktiven
Personalreferenzen, nicht bereits archivierte Personalzuordnungen.

## Migration und Kalender

Identische Root/Rental-Migration **054** und Startup-Konstante ergänzen nur
`job_employees.updated_at`, `deleted_at` und Lebenszyklus-/Versionsschutz.
Vorhandene Rollen und Verknüpfungen bleiben erhalten. Kein manueller Eingriff
in Produktionsdaten. Startup-DDL ist mit5s Lock-/30s Statement-Limit begrenzt.
Die DB verhindert Hard-Delete, Identitätswechsel, Änderungen archivierter Rollen
und neue/restaurierte Zuordnungen zu inaktiven Mitarbeitern. Aktive Mitarbeiter
werden mit FOR SHARE gegen parallele Deaktivierung gesperrt. Der bestehende
Job-Child-Guard schützt archivierte Jobs und versioniert alle Writer.

Die vorhandene, optional konfigurierte M365-Kalenderanbindung wird nach einem
erfolgreichen Commit best effort angestoßen. Der Entwurf weist dies aus.
Datenbankerfolg garantiert keine Graph-Zustellung; dafür gibt es hier keine
neue Outbox oder Versandgarantie. Archivierung entfernt alte Mitarbeitertermine
und aktualisiert die zentrale Teilnehmerliste. Erneuter regulärer Kalender-Sync
bereinigt auch verbliebene Legacy-Termine archivierter Zuordnungen, ohne diese
als Teilnehmer aufzunehmen. Receipt-Replay sendet keinen neuen Auftrag.

Rollout: zuerst RentalCore, dann MCP. Umgebungen, Volumes, Netzwerke, Ports und
andere Dienste erhalten. Anschließend Image/OCI-Quellrevision und Health/Ready
prüfen. MCP-Werkzeugliste neu laden und benötigte Aktionsscopes explizit freigeben.
Produktive Personalzuordnungen werden nicht als Tests verändert.
