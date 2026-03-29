DROP INDEX IF EXISTS "idx_document_packet_rules_resource_type";
DROP INDEX IF EXISTS "uq_document_packet_rules_resource_type_document_type";
DROP TABLE IF EXISTS "document_packet_rules";

ALTER TABLE "document_upload_sessions"
    DROP COLUMN IF EXISTS "lineage_id";

DROP INDEX IF EXISTS "uq_documents_lineage_version_number";
DROP INDEX IF EXISTS "uq_documents_current_lineage";
DROP INDEX IF EXISTS "idx_documents_current_version";
DROP INDEX IF EXISTS "idx_documents_lineage_id";

ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "storage_legal_hold",
    DROP COLUMN IF EXISTS "storage_retention_until",
    DROP COLUMN IF EXISTS "storage_retention_mode",
    DROP COLUMN IF EXISTS "storage_version_id",
    DROP COLUMN IF EXISTS "checksum_sha256",
    DROP COLUMN IF EXISTS "is_current_version",
    DROP COLUMN IF EXISTS "version_number",
    DROP COLUMN IF EXISTS "lineage_id";
