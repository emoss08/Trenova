--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A reviewer can reject a document (a blurred POD, a BOL for the wrong load) so it
-- stops counting toward a shipment's billing requirements. The rejection is an
-- accountability record like the approval beside it: who, when and why.
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "rejected_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rejected_at" bigint,
    ADD COLUMN IF NOT EXISTS "rejection_reason" text;

--bun:split
ALTER TABLE "documents"
    ADD CONSTRAINT "fk_documents_rejected_by" FOREIGN KEY ("rejected_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
UPDATE
    "documents"
SET
    "rejected_at" = "updated_at",
    "rejection_reason" = 'Rejected before rejection reasons were recorded'
WHERE
    "status" = 'Rejected'
    AND "rejected_at" IS NULL;

--bun:split
ALTER TABLE "documents"
    ADD CONSTRAINT "chk_documents_rejection_recorded" CHECK (
        ("status" = 'Rejected' AND "rejected_at" IS NOT NULL AND "rejection_reason" IS NOT NULL)
        OR ("status" <> 'Rejected' AND "rejected_by_id" IS NULL AND "rejected_at" IS NULL AND "rejection_reason" IS NULL)
    );
