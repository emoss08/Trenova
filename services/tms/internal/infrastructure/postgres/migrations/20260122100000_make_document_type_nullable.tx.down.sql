--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "fk_documents_document_type";

--bun:split
ALTER TABLE "documents"
    ALTER COLUMN "document_type_id" SET NOT NULL;

--bun:split
ALTER TABLE "documents"
    ADD CONSTRAINT "fk_documents_document_type"
    FOREIGN KEY ("document_type_id", "business_unit_id", "organization_id")
    REFERENCES "document_types" ("id", "business_unit_id", "organization_id")
    ON UPDATE NO ACTION ON DELETE CASCADE;
