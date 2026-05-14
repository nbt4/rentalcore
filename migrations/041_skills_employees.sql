CREATE TABLE skills (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT skills_name_unique UNIQUE (name)
);

CREATE TABLE employees (
    id           BIGSERIAL PRIMARY KEY,
    first_name   VARCHAR(100) NOT NULL,
    last_name    VARCHAR(100) NOT NULL,
    email        VARCHAR(255),
    phone        VARCHAR(50),
    mobile       VARCHAR(50),
    street       VARCHAR(255),
    house_number VARCHAR(20),
    zip          VARCHAR(20),
    city         VARCHAR(100),
    country      VARCHAR(100) NOT NULL DEFAULT 'Deutschland',
    date_of_birth DATE,
    iban         VARCHAR(50),
    notes        TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_email_unique UNIQUE (email)
);

CREATE TABLE employee_skills (
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id    BIGINT NOT NULL REFERENCES skills(id)    ON DELETE CASCADE,
    PRIMARY KEY (employee_id, skill_id)
);

-- Seed: Standard AV-Branche Skills
INSERT INTO skills (name, category) VALUES
  ('FOH-Mischung','Audio'),('Monitoring-Mischung','Audio'),('PA-System','Audio'),
  ('Mikrofonierung','Audio'),('Playback','Audio'),('Intercom','Audio'),
  ('Lichttechnik','Licht'),('Moving Lights','Licht'),('LED-Steuerung','Licht'),
  ('grandMA2','Licht'),('grandMA3','Licht'),('Haze / Fog','Licht'),('Followspot','Licht'),
  ('Projektionstechnik','Video'),('LED-Wall','Video'),('Kameratechnik','Video'),
  ('Video-Switching','Video'),('Live-Streaming','Video'),('Screen-Management','Video'),
  ('Rigging','Rigging'),('Anschlagmittel','Rigging'),('Traversensysteme','Rigging'),('Flugplanung','Rigging'),
  ('Bühnenaufbau','Bühne'),('Bühnenabbau','Bühne'),('Traversenbau','Bühne'),('Kabellegen','Bühne'),
  ('Projektmanagement','Projekt'),('Technische Leitung','Projekt'),('Veranstaltungsplanung','Projekt'),('CAD-Planung','Projekt'),
  ('Führerschein Klasse B','Fahrzeug'),('Führerschein Klasse BE','Fahrzeug'),
  ('Führerschein Klasse C','Fahrzeug'),('Gabelstapler','Fahrzeug'),('Hubarbeitsbühne','Fahrzeug');
