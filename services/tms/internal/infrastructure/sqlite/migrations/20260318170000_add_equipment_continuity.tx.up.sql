-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260318170000_add_equipment_continuity.tx.up.sql

CREATE TABLE IF NOT EXISTS equipment_continuity (
    "id" TEXT PRIMARY KEY,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "equipment_type" TEXT NOT NULL,
    "equipment_id" TEXT NOT NULL,
    "current_location_id" TEXT NOT NULL,
    "previous_continuity_id" TEXT,
    "source_type" TEXT NOT NULL,
    "source_shipment_id" TEXT,
    "source_shipment_move_id" TEXT,
    "source_assignment_id" TEXT,
    "is_current" INTEGER NOT NULL DEFAULT 1,
    "superseded_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_equipment_continuity_equipment
    ON equipment_continuity (organization_id, business_unit_id, equipment_type, equipment_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_equipment_continuity_shipment
    ON equipment_continuity (organization_id, business_unit_id, source_shipment_id);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_equipment_continuity_current_unique
    ON equipment_continuity (organization_id, business_unit_id, equipment_type, equipment_id)WHERE is_current = TRUE;
