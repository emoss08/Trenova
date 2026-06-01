CREATE TYPE "service_failure_reason_category_enum" AS ENUM(
    'Carrier',
    'Customer',
    'Facility',
    'Weather',
    'Equipment',
    'Documentation',
    'Other'
);

--bun:split
CREATE TYPE "service_failure_reason_applies_to_enum" AS ENUM(
    'Pickup',
    'Delivery',
    'Both'
);

--bun:split
CREATE TYPE "service_failure_type_enum" AS ENUM(
    'LatePickup',
    'LateDelivery'
);

--bun:split
CREATE TYPE "service_failure_source_enum" AS ENUM(
    'Detected',
    'Manual'
);

--bun:split
CREATE TYPE "service_failure_status_enum" AS ENUM(
    'Open',
    'Reviewed',
    'Resolved',
    'Voided'
);

--bun:split
CREATE TABLE IF NOT EXISTS "service_failure_reason_codes"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "code" varchar(64) NOT NULL,
    "label" varchar(120) NOT NULL,
    "description" text,
    "category" service_failure_reason_category_enum NOT NULL DEFAULT 'Carrier',
    "applies_to" service_failure_reason_applies_to_enum NOT NULL DEFAULT 'Both',
    "default_status_code" varchar(3),
    "default_reason_code" varchar(3),
    "default_exception_code" varchar(3),
    "default_note" text,
    "active" boolean NOT NULL DEFAULT TRUE,
    "sort_order" integer NOT NULL DEFAULT 100,
    "external_map" jsonb,
    "archived_at" bigint,
    "archived_by_id" varchar(100),
    "activated_at" bigint,
    "activated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_service_failure_reason_codes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sfrc_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sfrc_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sfrc_archived_by" FOREIGN KEY ("archived_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_sfrc_activated_by" FOREIGN KEY ("activated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_sfrc_x12_status_code_len" CHECK ("default_status_code" IS NULL OR length("default_status_code") BETWEEN 1 AND 3),
    CONSTRAINT "ck_sfrc_x12_reason_code_len" CHECK ("default_reason_code" IS NULL OR length("default_reason_code") BETWEEN 1 AND 3),
    CONSTRAINT "ck_sfrc_x12_exception_code_len" CHECK ("default_exception_code" IS NULL OR length("default_exception_code") BETWEEN 1 AND 3)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "ux_sfrc_tenant_code" ON "service_failure_reason_codes"(
    "organization_id",
    "business_unit_id",
    lower("code")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sfrc_tenant_active" ON "service_failure_reason_codes"(
    "organization_id",
    "business_unit_id",
    "active",
    "sort_order"
);

CREATE INDEX IF NOT EXISTS "idx_sfrc_applies_to" ON "service_failure_reason_codes"(
    "organization_id",
    "business_unit_id",
    "applies_to"
);

CREATE INDEX IF NOT EXISTS "idx_sfrc_external_map" ON "service_failure_reason_codes" USING GIN("external_map");

--bun:split
ALTER TABLE "service_failure_reason_codes"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("label", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("category"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("applies_to"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS "idx_sfrc_search" ON "service_failure_reason_codes" USING GIN("search_vector");

--bun:split
CREATE TABLE IF NOT EXISTS "service_failures"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "stop_id" varchar(100) NOT NULL,
    "reason_code_id" varchar(100),
    "number" varchar(64) NOT NULL,
    "type" service_failure_type_enum NOT NULL,
    "source" service_failure_source_enum NOT NULL DEFAULT 'Detected',
    "status" service_failure_status_enum NOT NULL DEFAULT 'Open',
    "stop_type" stop_type_enum NOT NULL,
    "scheduled_cutoff" bigint NOT NULL,
    "actual_arrival" bigint NOT NULL,
    "grace_period_minutes" integer NOT NULL DEFAULT 30,
    "late_minutes" bigint NOT NULL,
    "notes" text,
    "internal_notes" text,
    "x12_status_code_override" varchar(3),
    "x12_reason_code_override" varchar(3),
    "x12_exception_code" varchar(3),
    "detected_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "reviewed_at" bigint,
    "reviewed_by_id" varchar(100),
    "resolved_at" bigint,
    "resolved_by_id" varchar(100),
    "voided_at" bigint,
    "voided_by_id" varchar(100),
    "void_reason" text,
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_service_failures" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sf_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sf_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sf_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sf_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sf_stop" FOREIGN KEY ("stop_id", "organization_id", "business_unit_id") REFERENCES "stops"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sf_reason_code" FOREIGN KEY ("reason_code_id", "organization_id", "business_unit_id") REFERENCES "service_failure_reason_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_sf_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_sf_reviewed_by" FOREIGN KEY ("reviewed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_sf_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_sf_voided_by" FOREIGN KEY ("voided_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ux_sf_tenant_number" UNIQUE ("organization_id", "business_unit_id", "number"),
    CONSTRAINT "ck_sf_schedule_actual_positive" CHECK ("scheduled_cutoff" > 0 AND "actual_arrival" > 0),
    CONSTRAINT "ck_sf_late_minutes_positive" CHECK ("late_minutes" > 0),
    CONSTRAINT "ck_sf_grace_positive" CHECK ("grace_period_minutes" > 0),
    CONSTRAINT "ck_sf_review_state" CHECK (("reviewed_at" IS NULL AND "reviewed_by_id" IS NULL) OR ("reviewed_at" IS NOT NULL AND "reviewed_by_id" IS NOT NULL)),
    CONSTRAINT "ck_sf_resolve_state" CHECK (("resolved_at" IS NULL AND "resolved_by_id" IS NULL) OR ("resolved_at" IS NOT NULL AND "resolved_by_id" IS NOT NULL)),
    CONSTRAINT "ck_sf_void_state" CHECK (("voided_at" IS NULL AND "voided_by_id" IS NULL) OR ("voided_at" IS NOT NULL AND "voided_by_id" IS NOT NULL)),
    CONSTRAINT "ck_sf_terminal_timestamps" CHECK (
        ("status" = 'Reviewed' AND "reviewed_at" IS NOT NULL)
        OR ("status" = 'Resolved' AND "resolved_at" IS NOT NULL)
        OR ("status" = 'Voided' AND "voided_at" IS NOT NULL)
        OR "status" = 'Open'
    )
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "ux_sf_active_stop_type" ON "service_failures"(
    "shipment_id",
    "shipment_move_id",
    "stop_id",
    "organization_id",
    "business_unit_id",
    "type"
)
WHERE
    "status" IN ('Open', 'Reviewed');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sf_tenant_status_created" ON "service_failures"(
    "organization_id",
    "business_unit_id",
    "status",
    "created_at" DESC
);

CREATE INDEX IF NOT EXISTS "idx_sf_shipment_status" ON "service_failures"(
    "shipment_id",
    "organization_id",
    "business_unit_id",
    "status"
);

CREATE INDEX IF NOT EXISTS "idx_sf_stop" ON "service_failures"(
    "stop_id",
    "organization_id",
    "business_unit_id"
);

CREATE INDEX IF NOT EXISTS "idx_sf_reason_code" ON "service_failures"(
    "reason_code_id",
    "organization_id",
    "business_unit_id"
);

CREATE INDEX IF NOT EXISTS "idx_sf_detected_brin" ON "service_failures" USING BRIN("detected_at", "created_at") WITH (pages_per_range = 128);

--bun:split
ALTER TABLE "service_failures"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("number", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("type"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("source"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("notes", '')), 'C') ||
        setweight(immutable_to_tsvector('english', COALESCE("internal_notes", '')), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS "idx_sf_search" ON "service_failures" USING GIN("search_vector");

--bun:split
WITH defaults(code, label, description, category, applies_to, default_status_code, default_reason_code, default_exception_code, default_note, sort_order) AS (
    VALUES
        ('LATE_PICKUP', 'Late Pickup', 'Pickup completed after the scheduled cutoff and service failure grace period.', 'Carrier', 'Pickup', 'SD', 'NS', NULL, 'Late pickup service failure.', 10),
        ('LATE_DELIVERY', 'Late Delivery', 'Delivery completed after the scheduled cutoff and service failure grace period.', 'Carrier', 'Delivery', 'SD', 'NS', NULL, 'Late delivery service failure.', 20),
        ('FACILITY_DELAY', 'Facility Delay', 'Facility, dock, or appointment conditions caused the service failure.', 'Facility', 'Both', 'SD', 'NS', NULL, 'Facility-driven service failure.', 30),
        ('WEATHER_DELAY', 'Weather Delay', 'Weather conditions caused or contributed to the service failure.', 'Weather', 'Both', 'SD', 'NS', NULL, 'Weather-related service failure.', 40),
        ('EQUIPMENT_FAILURE', 'Equipment Failure', 'Power, trailer, or related equipment issue caused the service failure.', 'Equipment', 'Both', 'SD', 'NS', NULL, 'Equipment-related service failure.', 50),
        ('CUSTOMER_DELAY', 'Customer Delay', 'Customer action, availability, or instruction caused the service failure.', 'Customer', 'Both', 'SD', 'NS', NULL, 'Customer-driven service failure.', 60),
        ('DOCUMENT_DELAY', 'Documentation Delay', 'Documentation issue caused or contributed to the service failure.', 'Documentation', 'Both', 'SD', 'NS', NULL, 'Documentation-related service failure.', 70),
        ('OTHER_SERVICE_FAILURE', 'Other Service Failure', 'Operational service failure that does not fit another default reason.', 'Other', 'Both', 'SD', 'NS', NULL, 'Service failure.', 100)
)
INSERT INTO "service_failure_reason_codes"(
    "id",
    "organization_id",
    "business_unit_id",
    "code",
    "label",
    "description",
    "category",
    "applies_to",
    "default_status_code",
    "default_reason_code",
    "default_exception_code",
    "default_note",
    "active",
    "sort_order"
)
SELECT
    CONCAT('sfrc_', replace(gen_random_uuid()::text, '-', '')),
    org.id,
    org.business_unit_id,
    defaults.code,
    defaults.label,
    defaults.description,
    defaults.category::service_failure_reason_category_enum,
    defaults.applies_to::service_failure_reason_applies_to_enum,
    defaults.default_status_code,
    defaults.default_reason_code,
    defaults.default_exception_code,
    defaults.default_note,
    TRUE,
    defaults.sort_order
FROM "organizations" org
CROSS JOIN defaults
WHERE NOT EXISTS (
    SELECT 1
    FROM "service_failure_reason_codes" existing
    WHERE existing.organization_id = org.id
      AND existing.business_unit_id = org.business_unit_id
      AND lower(existing.code) = lower(defaults.code)
);
