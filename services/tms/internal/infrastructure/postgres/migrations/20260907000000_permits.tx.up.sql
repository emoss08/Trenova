CREATE TYPE "permit_status_enum" AS ENUM(
    'Pending',
    'Active',
    'Expired',
    'Void'
);

--bun:split
CREATE TYPE "permit_requirement_status_enum" AS ENUM(
    'Open',
    'Satisfied',
    'Superseded',
    'Waived'
);

--bun:split
CREATE TABLE IF NOT EXISTS "permits"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    "permit_number" varchar(100) NOT NULL,
    "status" permit_status_enum NOT NULL DEFAULT 'Pending',
    "issued_at" bigint,
    "expires_at" bigint,
    "cost" numeric(19, 4),
    "document_id" varchar(100),
    "notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_permits" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_permits_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_permits_dates" CHECK ("issued_at" IS NULL OR "expires_at" IS NULL OR "expires_at" > "issued_at"),
    CONSTRAINT "chk_permits_active_expiry" CHECK ("status" <> 'Active' OR "expires_at" IS NOT NULL),
    CONSTRAINT "chk_permits_cost" CHECK ("cost" IS NULL OR "cost" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_permits_shipment" ON "permits"("shipment_id", "state_id", "status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_permits_expiring" ON "permits"("organization_id", "business_unit_id", "expires_at")
WHERE
    "status" = 'Active';

--bun:split
ALTER TABLE "permits"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("permit_number", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("notes", '')), 'C')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_permits_search" ON "permits" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE permits IS 'Oversize and overweight permits the carrier holds for a shipment; recorded here after being obtained externally';

--bun:split
CREATE TABLE IF NOT EXISTS "permit_requirements"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    "status" permit_requirement_status_enum NOT NULL DEFAULT 'Open',
    "route_sequence" smallint NOT NULL DEFAULT 0,
    "exceedances" jsonb,
    "escorts" jsonb,
    "restrictions" jsonb,
    "lead_time_days" smallint NOT NULL DEFAULT 0,
    "validity_days" smallint NOT NULL DEFAULT 0,
    "estimated_fee" numeric(19, 4),
    "is_superload" boolean NOT NULL DEFAULT FALSE,
    "satisfied_by_permit_id" varchar(100),
    "waived_by_id" varchar(100),
    "waiver_reason" text,
    "provenance" jsonb,
    "derived_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_permit_requirements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_permit_requirements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permit_requirements_permit" FOREIGN KEY ("satisfied_by_permit_id", "organization_id", "business_unit_id") REFERENCES "permits"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_permit_requirements_waived_by" FOREIGN KEY ("waived_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_permit_requirements_satisfied" CHECK ("status" <> 'Satisfied' OR "satisfied_by_permit_id" IS NOT NULL),
    CONSTRAINT "chk_permit_requirements_waived" CHECK ("status" <> 'Waived' OR ("waiver_reason" IS NOT NULL AND char_length("waiver_reason") >= 10))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_permit_requirements_shipment" ON "permit_requirements"("shipment_id", "route_sequence");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_permit_requirements_open" ON "permit_requirements"("organization_id", "business_unit_id", "derived_at" DESC)
WHERE
    "status" = 'Open';

--bun:split
COMMENT ON TABLE permit_requirements IS 'Derived per shipment and jurisdiction from the travel envelope; recomputed on save and superseded rather than mutated. Provenance records the jurisdiction rule and its verification state, so a requirement built on unverified reference data says so';

--bun:split
INSERT INTO "hold_reasons"("id", "business_unit_id", "organization_id", "type", "code", "label", "description", "active", "default_severity", "default_blocks_dispatch", "default_blocks_delivery", "default_blocks_billing", "default_visible_to_customer", "sort_order")
SELECT
    'hr_permit_' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 20)),
    o."business_unit_id",
    o."id",
    'ComplianceHold',
    'PERMIT_REQUIRED',
    'Permit Required',
    'An oversize or overweight permit required by a jurisdiction on this route has not been recorded. Raised automatically and released once the permit is on file.',
    TRUE,
    'Blocking',
    TRUE,
    FALSE,
    FALSE,
    FALSE,
    10
FROM
    "organizations" o
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "accessorial_charges"("id", "business_unit_id", "organization_id", "code", "description", "status", "method", "rate_unit", "amount")
SELECT
    'acc_' || v."code" || '_' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 16)),
    o."business_unit_id",
    o."id",
    v."code",
    v."description",
    'Active',
    v."method"::accessorial_method_enum,
    v."rate_unit"::rate_unit_enum,
    v."amount"
FROM
    "organizations" o
    CROSS JOIN LATERAL (
        VALUES ('PERMIT', 'Oversize or overweight permit fee', 'Flat', NULL, 0),
        ('ESCORT', 'Pilot or escort vehicle mileage', 'PerUnit', 'Mile', 2.50),
        ('TARP', 'Tarping', 'Flat', NULL, 75),
        ('SECURE', 'Additional load securement', 'Flat', NULL, 50)) AS v("code",
        "description",
        "method",
        "rate_unit",
        "amount")
ON CONFLICT
    DO NOTHING;
