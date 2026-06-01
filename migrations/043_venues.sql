CREATE TABLE IF NOT EXISTS venues (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    street      VARCHAR(255),
    house_number VARCHAR(50),
    zip         VARCHAR(20),
    city        VARCHAR(255),
    contact_name VARCHAR(255),
    phone       VARCHAR(100),
    email       VARCHAR(255),
    notes       TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;
