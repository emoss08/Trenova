-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260906010000_rate_confirmations.tx.up.sql

CREATE TABLE IF NOT EXISTS "rate_confirmations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_assignment_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "revision" INTEGER NOT NULL DEFAULT 1,
    "status" TEXT NOT NULL DEFAULT 'Generated',
    "document_id" TEXT,
    "sent_at" INTEGER,
    "sent_to_emails" TEXT,
    "confirmed_at" INTEGER,
    "confirmed_by_name" TEXT,
    "voided_at" INTEGER,
    "void_reason" TEXT,
    "payload_snapshot" TEXT,
    "generated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_confirmations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_confirmations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_carrier_assignment" FOREIGN KEY ("carrier_assignment_id", "organization_id", "business_unit_id") REFERENCES "carrier_assignments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_confirmations_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_rate_confirmations_active_assignment" ON "rate_confirmations" ("carrier_assignment_id", "organization_id", "business_unit_id")WHERE
    "status" <> 'Voided';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_rate_confirmations_assignment_revision" ON "rate_confirmations" ("carrier_assignment_id", "revision", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_confirmations_shipment" ON "rate_confirmations" ("shipment_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_confirmations_carrier" ON "rate_confirmations" ("carrier_id", "organization_id");
