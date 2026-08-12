CREATE TABLE IF NOT EXISTS seed_rollbacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    seed_name VARCHAR(255) NOT NULL,
    seed_version VARCHAR(50) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    rolled_back_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    entities_deleted INT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_message TEXT
);
