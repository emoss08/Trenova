-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211014839_state.tx.up.sql

CREATE TABLE IF NOT EXISTS "us_states"
(
    "id" TEXT,
    "name" TEXT NOT NULL,
    "abbreviation" TEXT NOT NULL,
    "country_name" TEXT NOT NULL DEFAULT 'United States',
    "country_iso3" TEXT NOT NULL DEFAULT 'USA',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_us_states_name" ON "us_states" ("name");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_us_states_abbreviation" ON "us_states" ("abbreviation");
