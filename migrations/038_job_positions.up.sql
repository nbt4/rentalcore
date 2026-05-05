-- 038_job_positions.up.sql
-- Job positions: Angebots-artige Positionen pro Job (Produkte + Dienstleistungen)

CREATE TABLE IF NOT EXISTS job_positions (
    position_id     BIGSERIAL PRIMARY KEY,
    job_id          BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    position_type   VARCHAR(20) NOT NULL DEFAULT 'product' CHECK (position_type IN ('product', 'service')),
    product_id      INT REFERENCES products(productid) ON DELETE SET NULL,
    service_item_id BIGINT REFERENCES service_items(id) ON DELETE SET NULL,
    description     TEXT NOT NULL DEFAULT '',
    quantity        DECIMAL(10,2) NOT NULL DEFAULT 1,
    unit            VARCHAR(50) NOT NULL DEFAULT 'Stück',
    unit_price      DECIMAL(12,2) NOT NULL DEFAULT 0,
    follow_day_factor DECIMAL(4,2) NOT NULL DEFAULT 0.50,
    discount_percent  DECIMAL(5,2) NOT NULL DEFAULT 0,
    discount_amount   DECIMAL(12,2) NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_job_positions_job_id ON job_positions(job_id);
CREATE INDEX idx_job_positions_product_id ON job_positions(product_id);
CREATE INDEX idx_job_positions_service_item_id ON job_positions(service_item_id);

CREATE TABLE IF NOT EXISTS job_position_devices (
    id          BIGSERIAL PRIMARY KEY,
    position_id BIGINT NOT NULL REFERENCES job_positions(position_id) ON DELETE CASCADE,
    device_id   VARCHAR(50) NOT NULL,
    scanned_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scanned_by  VARCHAR(100) DEFAULT ''
);

CREATE INDEX idx_jpd_position ON job_position_devices(position_id);
CREATE INDEX idx_jpd_device ON job_position_devices(device_id);
CREATE UNIQUE INDEX idx_jpd_unique ON job_position_devices(position_id, device_id);
