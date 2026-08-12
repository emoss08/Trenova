-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260601130000_service_failures.tx.up.sql

CREATE TABLE IF NOT EXISTS "service_failure_reason_codes"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT NOT NULL DEFAULT 'Carrier',
    "applies_to" TEXT NOT NULL DEFAULT 'Both',
    "default_status_code" TEXT,
    "default_reason_code" TEXT,
    "default_exception_code" TEXT,
    "default_note" TEXT,
    "active" INTEGER NOT NULL DEFAULT 1,
    "sort_order" INTEGER NOT NULL DEFAULT 100,
    "external_map" TEXT,
    "archived_at" INTEGER,
    "archived_by_id" TEXT,
    "activated_at" INTEGER,
    "activated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "ux_sfrc_tenant_code" ON "service_failure_reason_codes" (
    "organization_id",
    "business_unit_id",
    lower("code")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sfrc_tenant_active" ON "service_failure_reason_codes" (
    "organization_id",
    "business_unit_id",
    "active",
    "sort_order"
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sfrc_applies_to" ON "service_failure_reason_codes" (
    "organization_id",
    "business_unit_id",
    "applies_to"
);

--bun:split

CREATE TABLE IF NOT EXISTS "service_failures"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "stop_id" TEXT NOT NULL,
    "reason_code_id" TEXT,
    "number" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "source" TEXT NOT NULL DEFAULT 'Detected',
    "status" TEXT NOT NULL DEFAULT 'Open',
    "stop_type" TEXT NOT NULL,
    "scheduled_cutoff" INTEGER NOT NULL,
    "actual_arrival" INTEGER NOT NULL,
    "grace_period_minutes" INTEGER NOT NULL DEFAULT 30,
    "late_minutes" INTEGER NOT NULL,
    "notes" TEXT,
    "internal_notes" TEXT,
    "x12_status_code_override" TEXT,
    "x12_reason_code_override" TEXT,
    "x12_exception_code" TEXT,
    "detected_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "reviewed_at" INTEGER,
    "reviewed_by_id" TEXT,
    "resolved_at" INTEGER,
    "resolved_by_id" TEXT,
    "voided_at" INTEGER,
    "voided_by_id" TEXT,
    "void_reason" TEXT,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
    CONSTRAINT "ck_sf_terminal_timestamps" CHECK (("status" = 'Reviewed' AND "reviewed_at" IS NOT NULL)
        OR ("status" = 'Resolved' AND "resolved_at" IS NOT NULL)
        OR ("status" = 'Voided' AND "voided_at" IS NOT NULL)
        OR "status" = 'Open')
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "ux_sf_active_stop_type" ON "service_failures" (
    "shipment_id",
    "shipment_move_id",
    "stop_id",
    "organization_id",
    "business_unit_id",
    "type"
)WHERE
    "status" IN ('Open', 'Reviewed');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sf_tenant_status_created" ON "service_failures" (
    "organization_id",
    "business_unit_id",
    "status",
    "created_at" DESC
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sf_shipment_status" ON "service_failures" (
    "shipment_id",
    "organization_id",
    "business_unit_id",
    "status"
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sf_stop" ON "service_failures" (
    "stop_id",
    "organization_id",
    "business_unit_id"
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sf_reason_code" ON "service_failures" (
    "reason_code_id",
    "organization_id",
    "business_unit_id"
);
