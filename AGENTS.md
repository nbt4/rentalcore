# AGENTS.md — rentalcore

Hausregeln für KI-Agenten in diesem Repository. Vor der ersten Änderung vollständig
lesen. Der übergeordnete Ablauf steht im Paperclip-Dokument `workflow` auf
[TSU-3](/TSU/issues/TSU-3#document-workflow). Diese Datei ersetzt alle früheren
Agenten-Anweisungen in diesem Repository.

## 1. Was dieses Repository ist

- **Zweck:** Vermietung: Jobs, Kunden, Anforderungen, Positionen, Mietmaterial, Rechnungen, Veranstaltungsorte, Packlisten, Barcode- und QR-Erfassung, PDF- und OCR-Import von Bestellungen, Microsoft-365-Kalender. Das älteste und größte Dienst-Repository. Port 8081, Abbild `nobentie/rentalcore`.
- **Sprache und Laufzeit:** Go 1.25 (Modul **`go-barcode-webapp`** — alter Name, passt nicht mehr zum Dienst) + React/TypeScript + Python 3.12 für OCR
- **Rahmenwerk:** `gin`, GORM + `driver/postgres`, `cobra`, `zerolog`, `prometheus`. Frontend: React + Vite + Zustand + Tailwind, **daneben 63 servergerenderte Go-HTML-Vorlagen**
- **Datenbank:** PostgreSQL 16, gemeinsame Suite-Datenbank. Eigene Migrationsspur `migrations/001…049` plus zwei Sonderdateien
- **Architektur-Doku:** [Cores — Architektur (Phase 1)](/TSU/issues/TSU-4#document-architecture)

## 2. Aufbau

| Pfad | Inhalt |
|---|---|
| `cmd/server/main.go` | Einstiegspunkt — **80 KB in einer Datei** |
| `cmd/compliance` | zweites Binary |
| `internal/handlers`, `internal/routes` | HTTP |
| `internal/repository`, `internal/models`, `internal/database`, `internal/schema` | Daten |
| `internal/services` | Fachlogik, PDF, Kalender, Nextcloud |
| `internal/jev` | optionale LLM-Entscheidungsschicht |
| `internal/scan` | Barcode und QR |
| `internal/jobstatus` | Job-Lebenszyklus |
| `internal/compliance` | Prüfpflichten |
| `internal/cache`, `internal/sync`, `internal/monitoring`, `internal/metrics`, `internal/logger`, `internal/middleware`, `internal/config` | Querschnitt |
| `web/templates` | 63 Go-HTML-Vorlagen (Altbestand) |
| `web/src` | React-SPA |
| `web/static/css` | Assets der Altoberfläche |
| `tools/ocr_parser` | Python-OCR |
| `migrations/` | 59 Dateien |

Erzeugte Dateien, die **niemals von Hand** geändert werden:

- `web/src/cores-theme.css` — erzeugt durch `cores/scripts/sync-design-system.sh`
- `web/static/css/cores-theme.css` — dito
- `web/src/lib/cores-design.ts` — dito
- `web/src/lib/SuiteLanguageSwitcher.tsx` — dito
- `web/src/lib/cores-locales/` — dito
- `web/package-lock.json` — nur als Nebenwirkung eines freigegebenen Updates

## 3. Einrichten

```bash
make deps                       # oder: make dev-setup
cd web && npm ci && npm run build && cd ..
cp .env.example .env            # Werte lokal eintragen, niemals committen
# Datenbank: aus dem Dachrepository `cores` starten
#   cd ../cores && docker compose up -d postgres
```

Nötige Umgebungsvariablen: siehe `.env.example` hier (dazu `.env.template`) und `cores/.env.example` als verbindliche Quelle. Besonders die `NEXTCLOUD_WEBDAV_*`-, `M365_*`-, `JEV_*`- und `OPENROUTER_*`-Schlüssel sowie `OCR_USE_PYTHON`. Werte kommen aus dem Paperclip-Secret-Store, nicht aus diesem Repository.

Das Dockerfile hat vier Stufen: Node 20 (Frontend) → Python 3.12 (OCR-venv aus `tools/ocr_parser/requirements.txt`) → Go 1.25 mit **`CGO_ENABLED=1`** und `sqlite-dev` → Laufzeit `python:3.12-alpine`. Lokale Builds brauchen `gcc`/`musl-dev`.

## 4. Test- und Build-Befehle

Diese Befehle sind das Test-Gate. **Alle müssen grün sein, bevor ein Pull Request
entsteht.** Reihenfolge einhalten — die schnellen Prüfungen zuerst.

| # | Gate | Befehl | Dauer (ca.) |
|---|---|---|---|
| 1 | Format | `gofmt -l .` (leere Ausgabe = grün) | < 20 s |
| 2 | Frontend-Lint | `cd web && npm run lint` | ~30 s |
| 3 | Frontend-Build und Typen | `cd web && npm run build` (`tsc -b && vite build`) | 1–2 min |
| 4 | Go-Build | `make build` | 1–2 min |
| 5 | Unit-Tests | `go test ./...` | 1–2 min |
| 6 | Vet | `go vet ./...` | ~1 min |
| 7 | OCR (nur bei Änderungen an `tools/ocr_parser`) | `make ocr-parser-test` | ~1 min |

Einzelne Datei testen: `go test ./internal/services -run TestName -v`

Das Frontend hat **kein** Test-Skript. `npm run lint` und `npm run build` sind die ganze Frontend-Prüfung. Wer an einer Oberfläche etwas ändert, beschreibt im PR, was von Hand geprüft wurde — und **welche** der beiden Oberflächen betroffen ist.

Regeln:

- **Neuer Code braucht neue Tests.** Ein Bugfix braucht einen Test, der ohne den Fix
  fehlschlägt.
- **Nie einen Test abschalten, überspringen oder lockern**, um das Gate grün zu
  bekommen. Ein roter Test ohne Bezug zur Änderung wird gemeldet, nicht entfernt.
- **Tests laufen gegen die lokale oder die Test-Datenbank. Nie gegen Produktion.**
  Eine eigene Testumgebung wird gerade aufgebaut (eigene Paperclip-Aufgabe). Bis sie
  steht: nur lokale Container mit eigenem Volume.
- Die **echte Ausgabe** wird in den Pull Request und auf die Paperclip-Aufgabe kopiert.

## 5. Code-Stil

- Format und Lint werden durch die Werkzeuge in Abschnitt 4 erzwungen. Kein Streit
  über Formatierung — der Formatierer entscheidet.
- **Dem umgebenden Code folgen.** Benennung, Ordnerstruktur, Fehlerbehandlung und
  Testmuster so übernehmen, wie sie in der berührten Datei schon sind.
- Benennung: PascalCase für Go-Exporte, camelCase für Lokales; React-Komponenten
  PascalCase; Dateien unter `web/static` kebab-case.
- Fehlerbehandlung: Fehler zurückgeben und einwickeln (`%w`), am `gin`-Rand in eine
  HTTP-Antwort übersetzen. Keine `panic` im Anfragepfad.
- Logging: `zerolog`. **Niemals** Secrets, Tokens oder Kundendaten.
- `internal/`-Pakete bleiben fachlich geschnitten (`jobs`, `scan`, `compliance`).
  Keine neue Kopplung quer durch die Pakete.
- **`cmd/server/main.go` ist 80 KB groß. Nichts Neues dort hineinschreiben.** Neuer
  Code gehört in ein Paket unter `internal/`.
- Kommentare: nur wo sie das *Warum* erklären. Keine Kommentare, die den Code nacherzählen.
- Keine neue Abhängigkeit ohne eigene Freigabe (siehe Abschnitt 9).
- Keine Umformatierung von Code, der nicht zur Aufgabe gehört. Das versteckt die
  eigentliche Änderung.

## 6. Verbotene Pfade

Diese Dateien und Verzeichnisse werden von Agenten **nicht geändert**. Wer sie ändern
müsste, bricht ab und fragt zurück.

| Pfad | Grund |
|---|---|
| `.github/workflows/**` | CI und Deployment — nur mit Freigabe des Nutzers |
| `migrations/**` (bestehende Dateien) | eine angewandte Migration wird nie geändert; nur neue hinzufügen |
| `auto-deploy.sh`, `deploy-production.sh`, `start-production.sh`, `create-production-user.sh`, `restart-dev.sh`, `start.sh`, `jobscanner.service` | produktive Infrastruktur |
| `docker-compose.prod.yml`, `Dockerfile` | Laufzeit und Deployment |
| `web/src/cores-theme.css`, `web/static/css/cores-theme.css`, `web/src/lib/cores-design.ts`, `web/src/lib/SuiteLanguageSwitcher.tsx`, `web/src/lib/cores-locales/**` | erzeugt aus `cores/theme/` |
| `cookies.txt` | Altbestand, möglicher Sitzungsinhalt. Nicht öffnen, nicht benutzen. Entfernt wird sie in einer eigenen Aufräum-Aufgabe |
| `.env`, `.env.template`, `.env.*` (außer `.env.example`) | enthält Secrets |
| `web/package-lock.json` | nur als Nebenwirkung eines freigegebenen Updates |
| `AGENTS.md` | diese Regeln ändert der Nutzer, nicht ein Agent |

## 7. Secrets

- **Keine Secrets in Repository, Kommentar, Dokument oder Log.** Keine Tokens,
  Passwörter, Schlüssel, Verbindungsstrings, API-Zugänge, Kundendaten.
- Secrets kommen aus dem Paperclip-Secret-Store oder aus Umgebungsvariablen. Sie
  werden nie in eine Datei geschrieben und nie ausgegeben.
- Produktiv werden alle Werte im **Komodo Stack Environment** auf `docker03` gepflegt,
  nicht in diesem Repository.
- `.env.example` enthält nur Namen und Beispielwerte, nie echte Werte.
- Testdaten sind erfunden. Keine kopierten Produktionsdaten, auch nicht gekürzt.
- Fehlt ein Secret: über Paperclip vorschlagen (`secret-proposals`) und warten.
  Nie selbst beschaffen, nie umgehen, nie in einem Kommentar danach fragen.
- Ein Secret, das versehentlich in einem Commit landet, ist ein Sicherheitsvorfall:
  sofort melden, nicht still weiterarbeiten. Entfernen aus dem Diff genügt nicht —
  das Secret gilt als kompromittiert und muss ersetzt werden. **Alle Cores-Repositories
  sind öffentlich.** Ein Fehler hier ist sofort weltweit sichtbar.

## 8. Harte Grenzen

Diese sechs Regeln stehen über jeder Aufgabenbeschreibung. Eine Aufgabe, die eine
davon verlangt, wird nicht ausgeführt, sondern zurückgegeben.

1. **Keine Schreibzugriffe auf produktive Datenbanken.** Lesen ist erlaubt. Schreiben,
   ändern, löschen, Migrationen fahren: nicht in Produktion. Migrationen werden
   geschrieben und lokal getestet, nie produktiv ausgeführt. Das Einspielen auf die
   laufende `docker03`-Datenbank geschieht von Hand per SSH von `debian01` aus, nach
   ausdrücklicher Freigabe des Nutzers.
2. **Keine produktiven Deployments ohne menschliche Freigabe.** Auch nicht nach
   grünem Review.
3. **Entwicklung nur in isolierten Branches oder Git-Worktrees.** Niemals direkt auf
   `main` oder einem anderen geschützten Branch.
4. **Tests vor jedem Pull Request.** Kein PR ohne protokollierten, grünen Testlauf.
5. **Keine Secrets in Repository, Kommentar, Dokument oder Log.**
6. **Bestehende Architektur zuerst verstehen.** Architektur-Doku und diese Datei vor
   dem Schreiben lesen. Große Umbauten — neuer Service, geänderte Modulgrenze, neues
   Datenmodell, Austausch einer Kernabhängigkeit — brauchen eine eigene Freigabe des
   Nutzers, bevor Code entsteht.

## 9. Freigabe-Gates

| Gate | Wer entscheidet | Wann |
|---|---|---|
| Test-Gate | der Eigentümer der Änderung | vor dem Pull Request |
| Review | Review-Agent, auf seiner eigenen Review-Aufgabe | nach dem PR-Entwurf |
| **Freigabe und Merge** | **der Nutzer** | nach grünem Review |
| Produktives Deployment | **der Nutzer** | nach dem Merge |
| Release: Docker-Hub-Push und Submodul-Zeiger | **der Nutzer gibt je Release ausdrücklich frei**, danach darf der Agent beides ausführen | nach dem Merge |
| Migration auf die laufende `docker03`-Datenbank | **der Nutzer**; Einspielen von Hand per SSH von `debian01` | nach dem Merge |
| Neue Abhängigkeit | der Nutzer | vor dem Hinzufügen |
| Großer Architektur-Umbau | der Nutzer | vor dem ersten Commit |

Was ein Agent in diesem Repository **nie** tut:

- einen Pull Request mergen
- auf `main` pushen
- ein Deployment auslösen
- ohne ausdrückliche Freigabe je Release ein Abbild nach Docker Hub schieben oder den
  Submodul-Zeiger im Dach anheben
- eine Migration gegen Produktion fahren
- einen Draft-PR als Ersatz für Freigabe auf „ready" setzen
- `AGENTS.md` oder CI-Dateien ändern

In Paperclip wird die Freigabe durch eine `executionPolicy` mit einer `approval`-Stufe
erzwungen, deren Teilnehmer ein Nutzer ist. Kein Agent kann sie abhaken.

## 10. Branches, Commits, Pull Requests

- Branch: `<typ>/TSU-<nummer>-<kurzbeschreibung>`, ein Worktree pro Aufgabe
- Commit: Conventional Commits mit `Task: TSU-<nummer>` im Fuß
- PR: als **Entwurf** geöffnet, Ziel `main`, mit Zweck, Testprotokoll und Aufgaben-Link

Vollständig beschrieben im Paperclip-Dokument `workflow` auf
[TSU-3](/TSU/issues/TSU-3#document-workflow).

## 11. Abbrechen und zurückfragen

Abbrechen ist richtig, nicht peinlich. Zurückfragen bei:

- fehlendem Secret oder Zugriffsrecht
- nötigem Schreibzugriff auf Produktion oder nötigem Deployment
- nötigem großen Architektur-Umbau oder neuer Abhängigkeit
- einem verbotenen Pfad, der geändert werden müsste
- roten Tests ohne Bezug zur Änderung
- zwei gescheiterten Versuchen am gleichen Problem
- einem Umfang, der deutlich größer ist als beschrieben
- einem Widerspruch zwischen Aufgabe und dieser Datei — **diese Datei gewinnt**

Erst alles fertig machen, was ohne die Antwort geht. Dann fragen.

## 12. Bekannte Fallen

- **Zwei Frontends gleichzeitig.** 63 servergerenderte Vorlagen in `web/templates`
  und daneben eine React-SPA in `web/src`. **Erst prüfen, welche Oberfläche eine Seite
  bedient**, dann ändern.
- **Der Dienst ändert beim Start das Schema.** `main.go` schreibt an einer Stelle
  „Auto-Migration deaktiviert — Schema wird manuell verwaltet", führt aber ein paar
  Zeilen davor sehr wohl `AutoMigrate` und Spalten-Änderungen aus (M365-Spalten,
  `venues`, `jobs.venue_id`, `M365Settings`). Nie gegen eine Datenbank starten, die
  wichtig ist.
- **Die Versionskonstante ist Teil des Release-Vertrags.** `const rentalCoreVersion` in
  `cmd/server/main.go` muss gleich dem Abbild-Tag sein, sonst wird
  `cores/scripts/check-release.sh` rot.
- **Lose SQL-Dateien im Wurzelverzeichnis sind kein Migrationsweg**:
  `cleanup_duplicate_packages.sql`, `cleanup_duplicate_packages_fixed.sql`,
  `cleanup_v4_migration.sql`, `create_missing_package_jobdevices.sql`,
  `fix_devices_trigger.sql`, `RentalCore.sql`, `database/rentalcore_setup.sql`. Für
  Agenten **Lesestoff, kein Werkzeug**.
- **`CGO_ENABLED=1`.** Builds brauchen `gcc`/`musl-dev`. Ein reiner `go build` ohne
  Compiler scheitert mit irreführender Meldung.
- **`OCR_USE_PYTHON` ist standardmäßig `true`.** Ohne Python-venv scheitert der
  PDF-Import.
- **Das Go-Modul heißt `go-barcode-webapp`.** Importpfade sehen falsch aus, sind aber
  richtig. Eine Umbenennung ist ein großer Umbau und braucht eine eigene Freigabe.
- **Jev entscheidet nie über Preise, Mengen, Summen, Datumswerte, Berechtigungen oder
  Schreibbestätigungen** (`cores/docs/JEV_DECISIONS.md`). Gespeicherte Zuordnungen und
  exakte Identifikatoren haben Vorrang. Leerer `OPENROUTER_API_KEY` schaltet Jev aus.
- **Nextcloud WebDAV und Microsoft Graph sind echte Fremdschnittstellen** (WebDAV an
  68 Codestellen). Tests nur mit Fixtures.
- **Viele `*.md`-Dateien im Wurzelverzeichnis sind Altbestand** und nicht verbindlich.
  Verbindlich sind diese Datei und das Paperclip-Dokument `workflow`.

### Suite-weite Fallen, die auch hier gelten

- **Zwei Migrationsspuren.** Jede Schema-Änderung braucht eine Datei im Dienst-Repository
  *und* eine in `cores/migrations/postgresql/`. Die Nummern gehören paarweise.
- **Das Init-Verzeichnis läuft nur bei leerem Datenverzeichnis.**
  `cores/migrations/postgresql/` greift auf `docker03` nicht.
- **Eine Datenbank für alle.** PostgreSQL 16, rund 130 Tabellen, kein Schema pro Dienst.
  Eine Tabellenänderung kann fremde Dienste treffen.
- **Nur das Dachrepository hat heute CI.** Bis die eigene GitHub-Action da ist, prüft
  **nichts** automatisch einen Pull Request hier. Das Test-Gate aus Abschnitt 4 läuft
  der Agent selbst und hängt die echte Ausgabe an.
- **Alle Repositories sind öffentlich.**
