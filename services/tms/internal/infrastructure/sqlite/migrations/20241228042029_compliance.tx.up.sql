-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241228042029_compliance.tx.up.sql

CREATE TABLE IF NOT EXISTS "hazmat_expirations" (
    "id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "years" INTEGER NOT NULL CHECK (years > 0),
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_hazmat_expirations" PRIMARY KEY ("id", "state_id"),
    CONSTRAINT "fk_hazmat_expirations_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_expirations_state" ON "hazmat_expirations" ("state_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_expirations_years" ON "hazmat_expirations" ("years");
