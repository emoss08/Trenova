-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260707120000_edi_inbound_pipeline.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_inbound_files"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "communication_profile_id" TEXT NOT NULL,
    "edi_partner_id" TEXT,
    "method" TEXT NOT NULL,
    "remote_path" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "checksum" TEXT NOT NULL,
    "size_bytes" INTEGER NOT NULL DEFAULT 0,
    "raw_content" TEXT NOT NULL,
    "interchange_control_number" TEXT,
    "isa_sender_qualifier" TEXT,
    "isa_sender_id" TEXT,
    "isa_receiver_qualifier" TEXT,
    "isa_receiver_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Received',
    "failure_reason" TEXT,
    "transaction_count" INTEGER NOT NULL DEFAULT 0,
    "received_at" INTEGER NOT NULL,
    "processed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_inbound_files" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_inbound_files_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_communication_profile" FOREIGN KEY ("communication_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_communication_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_inbound_files_checksum" ON "edi_inbound_files" ("organization_id", "business_unit_id", "communication_profile_id", "checksum");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_inbound_files_interchange" ON "edi_inbound_files" ("organization_id", "business_unit_id", "edi_partner_id", "interchange_control_number")WHERE
    "interchange_control_number" IS NOT NULL AND "edi_partner_id" IS NOT NULL AND "status" NOT IN ('Duplicate', 'Quarantined');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_status" ON "edi_inbound_files" ("organization_id", "business_unit_id", "status", "received_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_partner" ON "edi_inbound_files" ("edi_partner_id", "organization_id", "business_unit_id");

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "inbound_file_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_inbound_file" ON "edi_messages" ("inbound_file_id")WHERE
    "inbound_file_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_ack_lookup" ON "edi_messages" ("organization_id", "business_unit_id", "edi_partner_id", "direction", "group_control_number", "transaction_control_number");

--bun:split

ALTER TABLE "edi_load_tender_transfers" ADD COLUMN "inbound_message_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_inbound_message" ON "edi_load_tender_transfers" ("inbound_message_id")WHERE
    "inbound_message_id" IS NOT NULL;
