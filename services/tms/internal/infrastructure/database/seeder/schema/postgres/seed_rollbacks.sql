CREATE TABLE IF NOT EXISTS seed_rollbacks (
    id SERIAL PRIMARY KEY,
    seed_name VARCHAR(255) NOT NULL,
    seed_version VARCHAR(50) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    rolled_back_at TIMESTAMP NOT NULL DEFAULT NOW(),
    entities_deleted INT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_message TEXT
);
