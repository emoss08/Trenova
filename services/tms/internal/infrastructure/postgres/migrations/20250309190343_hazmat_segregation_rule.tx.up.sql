--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "segregation_type_enum" AS ENUM (
    'Prohibited',
    'Separated',
    'Distance',
    'Barrier'
);

-- bun:split
CREATE TABLE IF NOT EXISTS "hazmat_segregation_rules" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(100) NOT NULL,
    "description" text,
    "class_a" hazardous_class_enum NOT NULL,
    "class_b" hazardous_class_enum NOT NULL,
    "hazmat_a_id" varchar(100),
    "hazmat_b_id" varchar(100),
    "segregation_type" segregation_type_enum NOT NULL,
    "minimum_distance" float,
    "distance_unit" varchar(10),
    "has_exceptions" boolean NOT NULL DEFAULT FALSE,
    "exception_notes" text,
    "reference_code" varchar(100),
    "regulation_source" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_hazmat_segregation_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_hazmat_segregation_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_a" FOREIGN KEY ("hazmat_a_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_b" FOREIGN KEY ("hazmat_b_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_hazmat_segregation_rules_distance" CHECK ((segregation_type != 'Distance') OR (segregation_type = 'Distance' AND minimum_distance IS NOT NULL AND distance_unit IS NOT NULL)),
    CONSTRAINT "chk_hazmat_segregation_rules_exceptions" CHECK ((NOT has_exceptions) OR (has_exceptions AND exception_notes IS NOT NULL))
);

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_status" ON "hazmat_segregation_rules" ("status");

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_business_unit" ON "hazmat_segregation_rules" ("business_unit_id", "organization_id");

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_classes" ON "hazmat_segregation_rules" ("class_a", "class_b");

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_hazmats" ON "hazmat_segregation_rules" ("hazmat_a_id", "hazmat_b_id");

COMMENT ON TABLE hazmat_segregation_rules IS 'Stores rules for segregation of incompatible hazardous materials during transport';

-- bun:split
ALTER TABLE "hazmat_segregation_rules"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("reference_code", '')), 'C') ||
        setweight(immutable_to_tsvector('simple', COALESCE("regulation_source", '')), 'C')
    ) STORED;

-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_search ON hazmat_segregation_rules USING GIN (search_vector);

-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_dates_brin ON hazmat_segregation_rules USING BRIN (created_at, updated_at) WITH (pages_per_range = 128);

-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_active ON hazmat_segregation_rules (created_at DESC)
WHERE
    status != 'Inactive';

-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_trgm_name_description ON hazmat_segregation_rules USING gin ((name || ' ' || coalesce(description, '') || ' ' || coalesce(reference_code, '')) gin_trgm_ops);

-- bun:split
CREATE UNIQUE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_unique ON hazmat_segregation_rules (organization_id, business_unit_id, class_a, class_b, COALESCE(hazmat_a_id, ''), COALESCE(hazmat_b_id, ''));
