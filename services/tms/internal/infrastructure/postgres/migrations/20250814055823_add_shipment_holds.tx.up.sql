--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
-- Enums with documentation
CREATE TYPE "hold_type_enum" AS ENUM(
    'OperationalHold',
    -- Operational: e.g., missing/changed appointment, facility closed, dock issues
    'ComplianceHold',
    -- Compliance: OOS, HOS, CDL, permits, safety blocks
    'CustomerHold',
    -- Customer-requested pause/change
    'FinanceHold' -- Finance/credit gating; commonly blocks billing, not movement
);

CREATE TYPE "hold_severity_enum" AS ENUM(
    'Informational',
    -- FYI only; never blocks
    'Advisory',
    -- Warns; may block billing depending on flags
    'Blocking' -- Actively blocks movement and/or billing
);

CREATE TYPE "hold_source_enum" AS ENUM(
    'User',
    -- Manually created by a user
    'Rule',
    -- Rule/automation within the system
    'API',
    -- External API caller
    'ELD',
    -- ELD/telematics integration
    'EDI' -- EDI integration
);

--bun:split
CREATE TABLE IF NOT EXISTS "shipment_holds"(
    "id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "type" hold_type_enum NOT NULL,
    "severity" hold_severity_enum NOT NULL DEFAULT 'Advisory',
    "reason_code" varchar(100),
    "notes" text,
    "source" hold_source_enum NOT NULL DEFAULT 'User',
    -- Gating flags
    "blocks_dispatch" boolean NOT NULL DEFAULT FALSE,
    "blocks_delivery" boolean NOT NULL DEFAULT FALSE,
    "blocks_billing" boolean NOT NULL DEFAULT FALSE,
    -- Visibility & extensibility
    "visible_to_customer" boolean NOT NULL DEFAULT FALSE,
    "metadata" jsonb,
    -- Lifecycle
    "started_at" bigint NOT NULL CHECK ("started_at" > 0),
    "released_at" bigint,
    -- Audit
    "created_by_id" varchar(100),
    "released_by_id" varchar(100),
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "version" bigint NOT NULL DEFAULT 0,
    -- Constraints
    CONSTRAINT "pk_shipment_holds" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    -- Multi-tenant scoping integrity
    CONSTRAINT "fk_shipment_holds_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_holds_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure hold belongs to an existing shipment in the same org/BU
    CONSTRAINT "fk_shipment_holds_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Users
    CONSTRAINT "fk_shipment_holds_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_shipment_holds_released_by" FOREIGN KEY ("released_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    -- Consistency checks
    CONSTRAINT "ck_shipment_holds_released_ge_started" CHECK ("released_at" IS NULL OR "released_at" >= "started_at"),
    CONSTRAINT "ck_shipment_holds_blocking_requires_flags" CHECK ("severity" <> 'Blocking' OR ("blocks_dispatch" OR "blocks_delivery" OR "blocks_billing"))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_holds_bu_org" ON "shipment_holds"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_type" ON "shipment_holds"("type");

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_source" ON "shipment_holds"("source");

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_active_by_shipment" ON "shipment_holds"("shipment_id")
WHERE
    "released_at" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_started_brin" ON "shipment_holds" USING BRIN("started_at", "created_at") WITH (pages_per_range = 128);

-- A useful covering index for tenants filtering by BU/Org frequently
CREATE INDEX IF NOT EXISTS "idx_shipment_holds_bu_org_include" ON "shipment_holds"("business_unit_id", "organization_id") INCLUDE ("type", "severity", "started_at", "released_at");

-- Enforce at most one *active* hold of a given type per shipment (per tenant)
CREATE UNIQUE INDEX IF NOT EXISTS "ux_shipment_holds_active_by_type" ON "shipment_holds"("shipment_id", "organization_id", "business_unit_id", "type")
WHERE
    "released_at" IS NULL;

--bun:split
COMMENT ON TABLE shipment_holds IS 'Time-ranged holds applied to shipments. Multiple concurrent holds allowed; Blocking severity requires at least one gating flag.';

--bun:split
CREATE OR REPLACE FUNCTION shipment_holds_set_updated_at()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS shipment_holds_updated_at_trigger ON shipment_holds;

--bun:split
CREATE TRIGGER shipment_holds_updated_at_trigger
    BEFORE INSERT OR UPDATE ON shipment_holds
    FOR EACH ROW
    EXECUTE FUNCTION shipment_holds_set_updated_at();

--bun:split
-- Planner hints for common filters
ALTER TABLE shipment_holds
    ALTER COLUMN type SET STATISTICS 1000;

ALTER TABLE shipment_holds
    ALTER COLUMN severity SET STATISTICS 1000;

ALTER TABLE shipment_holds
    ALTER COLUMN shipment_id SET STATISTICS 1000;

--bun:split
CREATE OR REPLACE FUNCTION protect_shipment_holds()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF OLD.released_at IS NOT NULL THEN
        RAISE EXCEPTION 'Cannot modify released holds';
    END IF;
    IF NEW.id <> OLD.id OR NEW.shipment_id <> OLD.shipment_id OR NEW.organization_id <> OLD.organization_id OR NEW.business_unit_id <> OLD.business_unit_id THEN
        RAISE EXCEPTION 'Scope fields are immutable';
    END IF;
    IF NEW.type <> OLD.type THEN
        RAISE EXCEPTION 'Hold type is immutable; release and create a new hold to reclassify';
    END IF;
    IF NEW.source <> OLD.source THEN
        RAISE EXCEPTION 'Source is immutable';
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS protect_shipment_holds_tr ON shipment_holds;

--bun:split
CREATE TRIGGER protect_shipment_holds_tr
    BEFORE UPDATE ON shipment_holds
    FOR EACH ROW
    EXECUTE FUNCTION protect_shipment_holds();

