CREATE TYPE "edi_inbound_file_status_enum" AS ENUM(
    'Received',
    'Parsed',
    'Processed',
    'PartiallyProcessed',
    'Quarantined',
    'Duplicate'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_inbound_files"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "communication_profile_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100),
    "method" edi_connection_method_enum NOT NULL,
    "remote_path" text NOT NULL,
    "file_name" varchar(512) NOT NULL,
    "checksum" varchar(64) NOT NULL,
    "size_bytes" bigint NOT NULL DEFAULT 0,
    "raw_content" text NOT NULL,
    "interchange_control_number" varchar(20),
    "isa_sender_qualifier" varchar(4),
    "isa_sender_id" varchar(20),
    "isa_receiver_qualifier" varchar(4),
    "isa_receiver_id" varchar(20),
    "status" edi_inbound_file_status_enum NOT NULL DEFAULT 'Received',
    "failure_reason" text,
    "transaction_count" integer NOT NULL DEFAULT 0,
    "received_at" bigint NOT NULL,
    "processed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_edi_inbound_files" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_inbound_files_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_communication_profile" FOREIGN KEY ("communication_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_communication_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_inbound_files_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_inbound_files_checksum" ON "edi_inbound_files"("organization_id", "business_unit_id", "communication_profile_id", "checksum");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_inbound_files_interchange" ON "edi_inbound_files"("organization_id", "business_unit_id", "edi_partner_id", "interchange_control_number")
WHERE
    "interchange_control_number" IS NOT NULL AND "edi_partner_id" IS NOT NULL AND "status" NOT IN ('Duplicate', 'Quarantined');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_status" ON "edi_inbound_files"("organization_id", "business_unit_id", "status", "received_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_partner" ON "edi_inbound_files"("edi_partner_id", "organization_id", "business_unit_id");

--bun:split
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "inbound_file_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_messages_inbound_file" ON "edi_messages"("inbound_file_id")
WHERE
    "inbound_file_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_messages_ack_lookup" ON "edi_messages"("organization_id", "business_unit_id", "edi_partner_id", "direction", "group_control_number", "transaction_control_number");

--bun:split
ALTER TABLE "edi_messages"
    ALTER COLUMN "template_id" DROP NOT NULL;

--bun:split
ALTER TABLE "edi_messages"
    ALTER COLUMN "template_version_id" DROP NOT NULL;

--bun:split
ALTER TABLE "edi_messages"
    ALTER COLUMN "partner_document_profile_id" DROP NOT NULL;

--bun:split
ALTER TABLE "edi_load_tender_transfers"
    ALTER COLUMN "source_shipment_id" DROP NOT NULL;

--bun:split
ALTER TABLE "edi_load_tender_transfers"
    ADD COLUMN IF NOT EXISTS "inbound_message_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_inbound_message" ON "edi_load_tender_transfers"("inbound_message_id")
WHERE
    "inbound_message_id" IS NOT NULL;

