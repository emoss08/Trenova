ALTER TABLE "document_upload_sessions"
    DROP CONSTRAINT IF EXISTS "fk_document_upload_sessions_document";

ALTER TABLE "document_upload_sessions"
    ADD CONSTRAINT "fk_document_upload_sessions_document"
        FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id")
        ON DELETE NO ACTION;
