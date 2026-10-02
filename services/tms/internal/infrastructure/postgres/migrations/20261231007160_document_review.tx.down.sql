--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "chk_documents_rejection_recorded",
    DROP CONSTRAINT IF EXISTS "fk_documents_rejected_by";

--bun:split
ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "rejection_reason",
    DROP COLUMN IF EXISTS "rejected_at",
    DROP COLUMN IF EXISTS "rejected_by_id";
