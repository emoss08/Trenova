-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260127152045_add_seed_created_entities_table.up.sql

CREATE TABLE IF NOT EXISTS seed_created_entities (
    "id" INTEGER PRIMARY KEY,
    "seed_name" TEXT NOT NULL,
    "table_name" TEXT NOT NULL,
    "entity_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL,
    UNIQUE(seed_name, table_name, entity_id)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_seed_created_entities_seed_name ON seed_created_entities (seed_name);

--bun:split

CREATE INDEX IF NOT EXISTS idx_seed_created_entities_table_name ON seed_created_entities (table_name);

--bun:split

CREATE INDEX IF NOT EXISTS idx_seed_created_entities_entity_id ON seed_created_entities (entity_id);
