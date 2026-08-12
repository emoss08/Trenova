-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260209000001_dot_hazmat_references.tx.up.sql

CREATE TABLE IF NOT EXISTS dot_hazmat_references (
    "id" TEXT PRIMARY KEY,
    "un_number" TEXT NOT NULL,
    "proper_shipping_name" TEXT NOT NULL,
    "hazard_class" TEXT NOT NULL,
    "subsidiary_hazard" TEXT,
    "packing_group" TEXT,
    "special_provisions" TEXT,
    "packaging_exceptions" TEXT,
    "packaging_non_bulk" TEXT,
    "packaging_bulk" TEXT,
    "quantity_passenger" TEXT,
    "quantity_cargo" TEXT,
    "vessel_stowage" TEXT,
    "erg_guide" TEXT,
    "symbols" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_dot_hazmat_ref_un_number ON dot_hazmat_references (un_number);
