-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250309190343_hazmat_segregation_rule.tx.up.sql

CREATE TABLE IF NOT EXISTS "hazmat_segregation_rules" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "description" TEXT,
    "class_a" TEXT NOT NULL,
    "class_b" TEXT NOT NULL,
    "hazmat_a_id" TEXT,
    "hazmat_b_id" TEXT,
    "segregation_type" TEXT NOT NULL,
    "minimum_distance" TEXT,
    "distance_unit" TEXT,
    "has_exceptions" INTEGER NOT NULL DEFAULT 0,
    "exception_notes" TEXT,
    "reference_code" TEXT,
    "regulation_source" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_hazmat_segregation_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_hazmat_segregation_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_a" FOREIGN KEY ("hazmat_a_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_b" FOREIGN KEY ("hazmat_b_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_hazmat_segregation_rules_distance" CHECK ((segregation_type != 'Distance') OR (segregation_type = 'Distance' AND minimum_distance IS NOT NULL AND distance_unit IS NOT NULL)),
    CONSTRAINT "chk_hazmat_segregation_rules_exceptions" CHECK ((NOT has_exceptions) OR (has_exceptions AND exception_notes IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_status" ON "hazmat_segregation_rules" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_business_unit" ON "hazmat_segregation_rules" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_classes" ON "hazmat_segregation_rules" ("class_a", "class_b");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_hazmats" ON "hazmat_segregation_rules" ("hazmat_a_id", "hazmat_b_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_active ON hazmat_segregation_rules (created_at DESC)WHERE
    status != 'Inactive';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_unique ON hazmat_segregation_rules (organization_id, business_unit_id, class_a, class_b, COALESCE(hazmat_a_id, ''), COALESCE(hazmat_b_id, ''));
