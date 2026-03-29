ALTER TABLE "documents"
    ADD COLUMN "lineage_id" VARCHAR(100),
    ADD COLUMN "version_number" BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN "is_current_version" BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN "checksum_sha256" VARCHAR(64),
    ADD COLUMN "storage_version_id" VARCHAR(255),
    ADD COLUMN "storage_retention_mode" VARCHAR(50),
    ADD COLUMN "storage_retention_until" BIGINT,
    ADD COLUMN "storage_legal_hold" BOOLEAN NOT NULL DEFAULT false;

UPDATE "documents"
SET "lineage_id" = "id"
WHERE "lineage_id" IS NULL;

ALTER TABLE "documents"
    ALTER COLUMN "lineage_id" SET NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_documents_lineage_id" ON "documents"("lineage_id", "organization_id", "business_unit_id");
CREATE INDEX IF NOT EXISTS "idx_documents_current_version" ON "documents"("resource_type", "resource_id", "is_current_version");
CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_current_lineage"
    ON "documents"("lineage_id", "organization_id", "business_unit_id")
    WHERE "is_current_version" = true;
CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_lineage_version_number"
    ON "documents"("lineage_id", "organization_id", "business_unit_id", "version_number");

ALTER TABLE "document_upload_sessions"
    ADD COLUMN "lineage_id" VARCHAR(100);

CREATE TABLE IF NOT EXISTS "document_packet_rules" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "resource_type" VARCHAR(100) NOT NULL,
    "document_type_id" VARCHAR(100) NOT NULL,
    "required" BOOLEAN NOT NULL DEFAULT false,
    "allow_multiple" BOOLEAN NOT NULL DEFAULT false,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "expiration_required" BOOLEAN NOT NULL DEFAULT false,
    "expiration_warning_days" INTEGER NOT NULL DEFAULT 30,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_packet_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_packet_rules_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id")
        REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "ck_document_packet_rules_resource_type" CHECK (
        "resource_type" IN ('shipment', 'trailer', 'tractor', 'worker')
    ),
    CONSTRAINT "ck_document_packet_rules_expiration_warning_days" CHECK (
        "expiration_warning_days" >= 0
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_packet_rules_resource_type_document_type"
    ON "document_packet_rules"("resource_type", "document_type_id", "organization_id", "business_unit_id");
CREATE INDEX IF NOT EXISTS "idx_document_packet_rules_resource_type"
    ON "document_packet_rules"("resource_type", "organization_id", "business_unit_id");
