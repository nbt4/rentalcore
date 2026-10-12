# RentalCore Agent Rules

## Autonomie und dauerhafte Operationsfreigabe

Der Nutzer hat diese Freigabe am 12. Oktober 2026 ausdrücklich für das
Cores-Dachrepository und alle Cores-Dienst-Repositories erteilt. Dieser Abschnitt
hat bei widersprüchlichen Freigabe-, Draft-, Merge-, Deployment- oder
TSU-Vorgaben Vorrang. Er betrifft die Arbeit des Entwicklungsagenten; fachliche
Benutzerrechte, OAuth-Scopes und Bestätigungen in den Anwendungen bleiben gültig.

- Beauftragte Arbeiten eigenständig bis zum geprüften und veröffentlichten Ergebnis
  abschließen. Keine erneute Einzelfreigabe für die folgenden Operationsschritte
  verlangen, wenn sie zum Auftrag gehören.
- Änderungen committen und zu GitHub pushen; PRs erstellen, reviewen, freigeben,
  von Draft auf ready setzen und nach erfolgreichen Prüfungen selbst mergen.
  Ein menschlicher Merge oder eine separate Paperclip-Review-Aufgabe ist keine Pflicht.
  Für umfangreiche Änderungen ein unabhängiges Review durch einen Review-Agenten
  nutzen und konkrete Befunde vor dem Merge beheben.
- Auf Arbeitsbranches oder in Worktrees entwickeln. `main` nach erfolgreicher
  Validierung eigenständig aktualisieren, bevorzugt durch einen PR-Merge.
  Keine fremden Änderungen überschreiben oder ungeprüft force-pushen.
- Docker-Abbilder aus committed Quellcode bauen und nach Prüfung des zuletzt
  veröffentlichten Tags mit der nächsten fortlaufenden Version und `latest` nach
  Docker Hub pushen. Submodul-Zeiger, Versionspins und Release-Dokumentation synchron halten.
- Über die vorkonfigurierten Komodo- und SSH-Zugänge auf `docker03` eigenständig
  deployen, Images pullen und die betroffenen Dienste starten, stoppen oder neu starten.
  Laufzeitumgebung und Volumes erhalten; anschließend Image, Quellrevision und
  Healthchecks prüfen. Andere Dienste nur ändern, wenn der Auftrag es erfordert.
- Build-, CI-, Deployment-Konfiguration und `AGENTS.md` im Aufgabenumfang pflegen.
  Prüfungen dürfen dadurch nicht ausgeschaltet oder zur Verdeckung von Fehlern
  gelockert werden. Fehler selbst diagnostizieren und beheben; erst bei tatsächlich
  fehlendem Zugang, fehlender Information oder unklarem Umfang nachfragen.
- Tests passend zur Änderung durchführen und die echte Ausgabe protokollieren.
  Code- und Release-Gates weiterhin in der dokumentierten Reihenfolge ausführen;
  Go-Beweisläufe mit `-count=1`. Reine Dokumentations- oder `AGENTS.md`-Änderungen
  brauchen Diff-/Strukturprüfungen, keinen erneuten Anwendungsbuild oder Image-Release.
- Branches verständlich nach dem Thema benennen; Conventional Commits verwenden.
  Eine TSU-Nummer oder Paperclip-Aufgabe ist optional, kein Arbeits- oder Release-Gate.
- Bestehende Zugangsdaten nur über vorkonfigurierte Sessions, Laufzeitumgebungen
  oder Secret-Manager verwenden. Keine Secrets in Repository, PR, Kommentar oder Log
  schreiben; keine privaten Laufzeit- oder Produktionsdaten zu Testdaten machen.
- Destruktive Produktionsdatenänderungen und manuelle Schema-/Migrationseingriffe
  benötigen einen konkreten Auftrag dafür; die allgemeine Git-/Deployment-Freigabe
  ersetzt diesen Auftrag nicht. Datenbanktests ausschließlich lokal oder in einer
  isolierten Testumgebung ausführen. Große fachliche Umbauten bei unklarem Auftrag
  zunächst klären.

## Mandatory suite design contract

- Before any UI work, read `../docs/DESIGN_SYSTEM.md` and `../theme/README.md` in the `cores` umbrella, or the canonical documents in `github.com/nbt4/cores` for standalone work.
- `web/src/cores-theme.css`, `web/src/lib/cores-design.ts` and `web/static/css/cores-theme.css` are generated and must never be edited directly.
- Both the React client and legacy Go templates obey the same suite tokens. Orange/brown Rental-only UI accents, white “muted” text and independent scrollbar/select rules must not be reintroduced.
- Dashboard greetings must use `suiteGreeting()`. Shell geometry, typography, tables, forms, dropdowns and dashboards follow the common contract.
- Run the umbrella design check, frontend build and Go tests before release; update README documentation for visible behavior changes.
