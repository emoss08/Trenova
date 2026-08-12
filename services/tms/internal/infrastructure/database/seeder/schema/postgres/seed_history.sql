DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'seed_status_enum') THEN
        CREATE TYPE seed_status_enum AS ENUM ('Active', 'Inactive', 'Orphaned');
    END IF;
END $$;

--bun:split
CREATE TABLE IF NOT EXISTS seed_history (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    checksum VARCHAR(32) NOT NULL,
    applied_at BIGINT NOT NULL,
    applied_by VARCHAR(255) NOT NULL,
    status seed_status_enum NOT NULL DEFAULT 'Active',
    details JSONB,
    error TEXT,
    notes TEXT,
    duration_ms BIGINT,
    UNIQUE (name, version, environment)
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_seed_history_name ON seed_history(name);

--bun:split
CREATE INDEX IF NOT EXISTS idx_seed_history_environment ON seed_history(environment);

--bun:split
CREATE INDEX IF NOT EXISTS idx_seed_history_applied_at ON seed_history(applied_at);

--bun:split
CREATE INDEX IF NOT EXISTS idx_seed_history_status ON seed_history(status);
