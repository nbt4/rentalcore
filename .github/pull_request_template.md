## Aufgabe

TSU-<nummer> — <Titel der Aufgabe>

## Was sich ändert

<Zwei bis fünf Sätze. Was tut der Code jetzt, was er vorher nicht tat.>

## Warum

<Der Grund. Welches Problem war da.>

## Umfang

- Berührte Module/Services:
- Datenbank-Migration enthalten: ja / nein
- Öffentliche API geändert: ja / nein
- Abhängigkeit hinzugefügt oder gehoben: ja / nein (wenn ja: welche und warum)

## Testlauf

<Befehle und Ergebnis, kopiert aus dem echten Lauf. Keine Zusammenfassung aus dem Kopf.>

| Gate | Befehl | Ergebnis |
|---|---|---|
| Lint | | |
| Typen | | |
| Unit | | |
| Integration (betroffene) | | |
| Build | | |

## Risiko und Rückweg

- Risiko: niedrig / mittel / hoch, mit einem Satz Begründung
- Rückweg: <wie man die Änderung zurückdreht>

## Leitplanken-Prüfung

- [ ] Keine Schreibzugriffe auf produktive Datenbanken
- [ ] Kein produktives Deployment ausgelöst
- [ ] Nur im isolierten Branch/Worktree gearbeitet
- [ ] Tests vor diesem PR gelaufen und grün
- [ ] Keine Secrets in Code, Commit, Log oder diesem PR
- [ ] Kein großer Architektur-Umbau ohne eigene Freigabe

## Offene Punkte

<Was der Reviewer und der Nutzer wissen müssen. "Keine" ist eine erlaubte Antwort.>
