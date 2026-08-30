BEGIN;

LOCK TABLE status IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE jobs IN SHARE ROW EXCLUSIVE MODE;

ALTER TABLE status DROP CONSTRAINT IF EXISTS chk_job_status_canonical;

INSERT INTO status (statusid, status, description, color, sort_order) VALUES
    (1, '__job_status_planning__', '', '#6c757d', 1),
    (2, '__job_status_confirmed__', '', '#17a2b8', 2),
    (4, '__job_status_completed__', '', '#007bff', 3),
    (6, '__job_status_cancelled__', '', '#dc3545', 4)
ON CONFLICT (statusid) DO NOTHING;

UPDATE jobs j
SET statusid = CASE
    WHEN LOWER(TRIM(s.status)) IN ('vorbereitung', 'bestätigt', 'bestaetigt', 'confirmed', 'aktiv', 'active', 'open', 'in progress', 'in_progress', 'pausiert', 'on hold') THEN 2
    WHEN LOWER(TRIM(s.status)) IN ('abgeschlossen', 'completed', 'abgerechnet', 'paid') THEN 4
    WHEN LOWER(TRIM(s.status)) IN ('storniert', 'cancelled', 'canceled') THEN 6
    ELSE 1
END
FROM status s
WHERE j.statusid = s.statusid;

DELETE FROM status WHERE statusid NOT IN (1, 2, 4, 6);

UPDATE status SET status='Planung', description='Job ist in Planung und noch nicht zur Ausgabe freigegeben', color='#6c757d', sort_order=1 WHERE statusid=1;
UPDATE status SET status='Bestätigt', description='Job ist verbindlich und zur Vorbereitung und Ausgabe freigegeben', color='#17a2b8', sort_order=2 WHERE statusid=2;
UPDATE status SET status='Abgeschlossen', description='Job ist beendet; ausgegebene Geräte befinden sich im Rücklauf', color='#007bff', sort_order=3 WHERE statusid=4;
UPDATE status SET status='Storniert', description='Job wurde storniert; Reservierungen sind aufgehoben', color='#dc3545', sort_order=4 WHERE statusid=6;

ALTER TABLE status ADD CONSTRAINT chk_job_status_canonical CHECK (
    (statusid=1 AND status='Planung') OR
    (statusid=2 AND status='Bestätigt') OR
    (statusid=4 AND status='Abgeschlossen') OR
    (statusid=6 AND status='Storniert')
);

SELECT setval('status_statusid_seq', GREATEST((SELECT MAX(statusid) FROM status), 1), true);

COMMIT;
