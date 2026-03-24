--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Statement # 1
-- Enums with documentation
CREATE TYPE "segregation_type_enum" AS ENUM (
    'Prohibited',
    'Separated',
    'Distance',
    'Barrier'
);

-- Statement # 2
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
    -- Constraints
    CONSTRAINT "pk_hazmat_segregation_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_hazmat_segregation_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_a" FOREIGN KEY ("hazmat_a_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_hazmat_segregation_rules_hazmat_b" FOREIGN KEY ("hazmat_b_id", "organization_id", "business_unit_id") REFERENCES "hazardous_materials" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_hazmat_segregation_rules_distance" CHECK ((segregation_type != 'Distance') OR (segregation_type = 'Distance' AND minimum_distance IS NOT NULL AND distance_unit IS NOT NULL)),
    CONSTRAINT "chk_hazmat_segregation_rules_exceptions" CHECK ((NOT has_exceptions) OR (has_exceptions AND exception_notes IS NOT NULL))
);

-- Statement # 3
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_status" ON "hazmat_segregation_rules" ("status");

-- Statement # 4
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_business_unit" ON "hazmat_segregation_rules" ("business_unit_id", "organization_id");

-- Statement # 5
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_classes" ON "hazmat_segregation_rules" ("class_a", "class_b");

-- Statement # 6
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_hazmat_segregation_rules_hazmats" ON "hazmat_segregation_rules" ("hazmat_a_id", "hazmat_b_id");

-- Statement # 7
COMMENT ON TABLE hazmat_segregation_rules IS 'Stores rules for segregation of incompatible hazardous materials during transport';

-- Statement # 8
-- bun:split
ALTER TABLE "hazmat_segregation_rules"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Statement # 9
-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_search ON hazmat_segregation_rules USING GIN (search_vector);

-- Statement # 10
-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_dates_brin ON hazmat_segregation_rules USING BRIN (created_at, updated_at) WITH (pages_per_range = 128);

-- Statement # 11
-- bun:split
CREATE OR REPLACE FUNCTION hazmat_segregation_rules_search_vector_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', coalesce(NEW.name, '')), 'A') || setweight(to_tsvector('simple', coalesce(NEW.description, '')), 'B') || setweight(to_tsvector('simple', coalesce(NEW.reference_code, '')), 'C') || setweight(to_tsvector('simple', coalesce(NEW.regulation_source, '')), 'C');
    -- Auto-update timestamps
    NEW.updated_at := extract(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

-- Statement # 12
--bun:split
DROP TRIGGER IF EXISTS hazmat_segregation_rules_search_vector_trigger ON hazmat_segregation_rules;

-- Statement # 13
--bun:split
CREATE TRIGGER hazmat_segregation_rules_search_vector_trigger
    BEFORE INSERT OR UPDATE ON hazmat_segregation_rules
    FOR EACH ROW
    EXECUTE FUNCTION hazmat_segregation_rules_search_vector_update ();

-- Statement # 14
-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_active ON hazmat_segregation_rules (created_at DESC)
WHERE
    status != 'Inactive';

-- Statement # 15
-- bun:split
ALTER TABLE hazmat_segregation_rules
    ALTER COLUMN status SET STATISTICS 1000;

-- Statement # 16
-- bun:split
ALTER TABLE hazmat_segregation_rules
    ALTER COLUMN organization_id SET STATISTICS 1000;

-- Statement # 17
-- bun:split
ALTER TABLE hazmat_segregation_rules
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

-- Statement # 18
-- bun:split
ALTER TABLE hazmat_segregation_rules
    ALTER COLUMN class_a SET STATISTICS 1000;

-- Statement # 19
-- bun:split
ALTER TABLE hazmat_segregation_rules
    ALTER COLUMN class_b SET STATISTICS 1000;

-- Statement # 20
-- bun:split
CREATE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_trgm_name_description ON hazmat_segregation_rules USING gin ((name || ' ' || coalesce(description, '') || ' ' || coalesce(reference_code, '')) gin_trgm_ops);

-- Statement # 21
-- bun:split
-- Create a unique index to prevent duplicate rules for the same classes/materials
CREATE UNIQUE INDEX IF NOT EXISTS idx_hazmat_segregation_rules_unique ON hazmat_segregation_rules (organization_id, business_unit_id, class_a, class_b, COALESCE(hazmat_a_id, ''), COALESCE(hazmat_b_id, ''));

