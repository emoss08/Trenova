-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260514120000_edi_connections_and_profiles.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_connections"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source_organization_id" TEXT NOT NULL,
    "target_organization_id" TEXT NOT NULL,
    "source_partner_id" TEXT,
    "target_partner_id" TEXT,
    "method" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'PendingAcceptance',
    "capabilities" TEXT NOT NULL DEFAULT '{}',
    "source_partner_config" TEXT NOT NULL DEFAULT '{}',
    "target_partner_config" TEXT NOT NULL DEFAULT '{}',
    "requested_by_id" TEXT,
    "requested_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "accepted_by_id" TEXT,
    "accepted_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejected_at" INTEGER,
    "rejection_reason" TEXT,
    "suspended_by_id" TEXT,
    "suspended_at" INTEGER,
    "revoked_by_id" TEXT,
    "revoked_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_connections" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_connections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_connections_source_org" FOREIGN KEY ("source_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_connections_target_org" FOREIGN KEY ("target_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_connections_distinct_orgs" CHECK ("source_organization_id" <> "target_organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_connections_source"
    ON "edi_connections" ("source_organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_connections_target"
    ON "edi_connections" ("target_organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "edi_communication_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_connection_id" TEXT,
    "edi_partner_id" TEXT,
    "method" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "description" TEXT,
    "config" TEXT NOT NULL DEFAULT '{}',
    "encrypted_secrets" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_communication_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_communication_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_communication_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_communication_profiles_connection" FOREIGN KEY ("edi_connection_id") REFERENCES "edi_connections"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_communication_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_communication_profiles_name_org"
    ON "edi_communication_profiles" (lower("name"), "business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_communication_profiles_partner"
    ON "edi_communication_profiles" ("edi_partner_id", "business_unit_id", "organization_id", "status");

--bun:split

ALTER TABLE "edi_partners" ADD COLUMN "edi_connection_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "tender_status" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "entry_method" TEXT NOT NULL DEFAULT 'Manual';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_tender_status"
    ON "shipments" ("business_unit_id", "organization_id", "tender_status")WHERE "tender_status" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_entry_method"
    ON "shipments" ("business_unit_id", "organization_id", "entry_method");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_shipment_links"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source_organization_id" TEXT NOT NULL,
    "target_organization_id" TEXT NOT NULL,
    "source_shipment_id" TEXT NOT NULL,
    "target_shipment_id" TEXT NOT NULL,
    "tender_transfer_id" TEXT NOT NULL,
    "originating_message_id" TEXT,
    "sync_policy" TEXT NOT NULL DEFAULT 'AutoOperational',
    "field_ownership" TEXT NOT NULL DEFAULT '{}',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_shipment_links" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_shipment_links_source_org" FOREIGN KEY ("source_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_target_org" FOREIGN KEY ("target_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_source_shipment" FOREIGN KEY ("source_shipment_id", "business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_target_shipment" FOREIGN KEY ("target_shipment_id", "business_unit_id", "target_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_transfer" FOREIGN KEY ("tender_transfer_id") REFERENCES "edi_load_tender_transfers"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_shipment_links_transfer"
    ON "edi_shipment_links" ("tender_transfer_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_shipment_links_source_target"
    ON "edi_shipment_links" ("business_unit_id", "source_organization_id", "source_shipment_id", "target_organization_id", "target_shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_shipment_links_source_lookup"
    ON "edi_shipment_links" ("business_unit_id", "source_organization_id", "source_shipment_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_shipment_links_target_lookup"
    ON "edi_shipment_links" ("business_unit_id", "target_organization_id", "target_shipment_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_transfer_changes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_link_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "change_type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'PendingReview',
    "conflict_status" TEXT NOT NULL DEFAULT 'None',
    "conflict_reason" TEXT,
    "idempotency_key" TEXT NOT NULL,
    "source_shipment_version" INTEGER NOT NULL,
    "target_shipment_version" INTEGER NOT NULL,
    "payload" TEXT NOT NULL DEFAULT '{}',
    "diff" TEXT NOT NULL DEFAULT '{}',
    "reviewed_by_id" TEXT,
    "reviewed_at" INTEGER,
    "applied_by_id" TEXT,
    "applied_at" INTEGER,
    "failure_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_transfer_changes" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_transfer_changes_link" FOREIGN KEY ("shipment_link_id", "business_unit_id") REFERENCES "edi_shipment_links"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_transfer_changes_reviewed_by" FOREIGN KEY ("reviewed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_transfer_changes_applied_by" FOREIGN KEY ("applied_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transfer_changes_idempotency"
    ON "edi_transfer_changes" ("shipment_link_id", "business_unit_id", "direction", "change_type", "idempotency_key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_transfer_changes_link_status"
    ON "edi_transfer_changes" ("shipment_link_id", "business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_transfer_changes_conflict"
    ON "edi_transfer_changes" ("business_unit_id", "conflict_status", "status", "created_at" DESC);
