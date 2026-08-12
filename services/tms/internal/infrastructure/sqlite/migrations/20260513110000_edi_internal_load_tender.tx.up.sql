-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260513110000_edi_internal_load_tender.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_mapping_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_mapping_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_mapping_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_mapping_profiles_partner"
    ON "edi_mapping_profiles" ("edi_partner_id", "business_unit_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_mapping_profile_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "mapping_profile_id" TEXT NOT NULL,
    "entity_type" TEXT NOT NULL,
    "source_id" TEXT NOT NULL,
    "source_label" TEXT,
    "target_id" TEXT NOT NULL,
    "target_label" TEXT,
    "created_by_id" TEXT,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_mapping_profile_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_mapping_profile_items_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_mapping_profile_items_profile" FOREIGN KEY ("mapping_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_mapping_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_mapping_profile_items_unique"
    ON "edi_mapping_profile_items" ("edi_partner_id", "business_unit_id", "organization_id", "entity_type", "source_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_mapping_profile_items_target"
    ON "edi_mapping_profile_items" ("target_id", "entity_type", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_load_tender_transfers"(
    "id" TEXT NOT NULL,
    "source_organization_id" TEXT NOT NULL,
    "source_business_unit_id" TEXT NOT NULL,
    "target_organization_id" TEXT NOT NULL,
    "target_business_unit_id" TEXT NOT NULL,
    "source_partner_id" TEXT NOT NULL,
    "target_partner_id" TEXT NOT NULL,
    "source_shipment_id" TEXT NOT NULL,
    "target_shipment_id" TEXT,
    "status" TEXT NOT NULL,
    "tender_payload" TEXT NOT NULL,
    "mapping_snapshot" TEXT NOT NULL DEFAULT '[]',
    "rejection_reason" TEXT,
    "failure_reason" TEXT,
    "submitted_by_id" TEXT,
    "submitted_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejected_at" INTEGER,
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_load_tender_transfers" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_load_tender_transfers_source_partner" FOREIGN KEY ("source_partner_id", "source_business_unit_id", "source_organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_target_partner" FOREIGN KEY ("target_partner_id", "target_business_unit_id", "target_organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_source_shipment" FOREIGN KEY ("source_shipment_id", "source_business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_target_shipment" FOREIGN KEY ("target_shipment_id", "target_business_unit_id", "target_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_open_unique"
    ON "edi_load_tender_transfers" ("source_shipment_id", "source_partner_id")WHERE "status" NOT IN ('Approved', 'Rejected', 'Canceled', 'Failed');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_inbound"
    ON "edi_load_tender_transfers" ("target_organization_id", "target_business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_outbound"
    ON "edi_load_tender_transfers" ("source_organization_id", "source_business_unit_id", "status", "created_at" DESC);
