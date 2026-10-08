# Terminradar im RentalCore-Dashboard

Die React-SPA lädt Jobs und Kontakte beim Öffnen des Dashboards. Solange der Tab
sichtbar ist, lädt sie die Daten alle 60 Sekunden erneut. Beim Zurückkehren zum
Tab oder Fenster erfolgt eine sofortige Aktualisierung. Hintergrundaktualisierungen
lassen die bisherigen Daten sichtbar; gleichzeitig läuft höchstens ein Ladevorgang.
Beim Verlassen des Dashboards werden Timer, Ereignis-Listener und laufende Anfragen
aufgeräumt.

Der Terminradar zeigt höchstens fünf laufende oder kommende offene Jobs in
Datumsreihenfolge. Ein Enddatum vor dem heutigen lokalen Kalendertag schließt den
Job aus dem Radar aus. Überfällige Jobs bleiben im Bereich „Jetzt bearbeiten“ und
im Überfälligkeitszähler sichtbar. Abgeschlossene, stornierte und nicht terminierte
Jobs erscheinen nicht im Radar. Der lokale Kalendertag wird bei jeder
Aktualisierung neu bestimmt, damit auch ein über Mitternacht geöffnetes Dashboard
seine Termine und Tageskennzahlen aktualisiert.

Bei einem Ladefehler wird auch die andere laufende Anfrage abgebrochen, bevor
ein weiterer Ladevorgang starten kann. Die letzten Daten bleiben sichtbar. Die bestehende
Fehlermeldung bietet „Erneut versuchen“; die nächste automatische Aktualisierung
kann den Fehler ebenfalls beheben.

Die Regressionstests benötigen Node 20 oder neuer und die vorhandenen
Frontend-Abhängigkeiten:

```sh
cd web
node --test tests/dashboard.test.mjs
```

Sie prüfen Terminfilterung, Sortierung, Tagesgrenzen, Aktualisierungsintervalle,
Tab-Sichtbarkeit und das Entfernen der Listener und Timer. Das
[Testprotokoll vom 8. Oktober 2026](validation/terminradar-2026-10-08.md)
enthält zusätzlich den Browservergleich mit der unveränderten Version.
