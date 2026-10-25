ALTER TABLE "documents"
    ADD COLUMN "lineage_id" varchar(100),
    ADD COLUMN "version_number" bigint NOT NULL DEFAULT 1,
    ADD COLUMN "is_current_version" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN "checksum_sha256" varchar(64),
    ADD COLUMN "storage_version_id" varchar(255),
    ADD COLUMN "storage_retention_mode" varchar(50),
    ADD COLUMN "storage_retention_until" bigint,
    ADD COLUMN "storage_legal_hold" boolean NOT NULL DEFAULT FALSE;

UPDATE
    "documents"
SET
    "lineage_id" = "id"
WHERE
    "lineage_id" IS NULL;

ALTER TABLE "documents"
    ALTER COLUMN "lineage_id" SET NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_documents_lineage_id" ON "documents"("lineage_id", "organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_documents_current_version" ON "documents"("resource_type", "resource_id", "is_current_version");

CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_current_lineage" ON "documents"("lineage_id", "organization_id", "business_unit_id")
WHERE
    "is_current_version" = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_lineage_version_number" ON "documents"("lineage_id", "organization_id", "business_unit_id", "version_number");

ALTER TABLE "document_upload_sessions"
    ADD COLUMN "lineage_id" varchar(100);

CREATE TYPE "document_packet_rule_resource_type" AS ENUM(
    'Shipment',
    'Trailer',
    'Tractor',
    'Worker'
);

CREATE TABLE IF NOT EXISTS "document_packet_rules"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "resource_type" "document_packet_rule_resource_type" NOT NULL DEFAULT 'Shipment',
    "document_type_id" varchar(100) NOT NULL,
    "required" boolean NOT NULL DEFAULT FALSE,
    "allow_multiple" boolean NOT NULL DEFAULT FALSE,
    "display_order" integer NOT NULL DEFAULT 0,
    "expiration_required" boolean NOT NULL DEFAULT FALSE,
    "expiration_warning_days" integer NOT NULL DEFAULT 30,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_document_packet_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_packet_rules_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "ck_document_packet_rules_expiration_warning_days" CHECK ("expiration_warning_days" >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_packet_rules_resource_type_document_type" ON "document_packet_rules"("resource_type", "document_type_id", "organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_document_packet_rules_resource_type" ON "document_packet_rules"("resource_type", "organization_id", "business_unit_id");

