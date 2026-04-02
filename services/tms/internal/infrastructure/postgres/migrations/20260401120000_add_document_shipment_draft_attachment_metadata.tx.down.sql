ALTER TABLE "document_shipment_drafts"
    DROP COLUMN IF EXISTS "attached_by_id",
    DROP COLUMN IF EXISTS "attached_at",
    DROP COLUMN IF EXISTS "attached_shipment_id";
