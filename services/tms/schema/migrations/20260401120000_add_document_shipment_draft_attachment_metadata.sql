ALTER TABLE "document_shipment_drafts"
    ADD COLUMN IF NOT EXISTS "attached_shipment_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "attached_at" bigint,
    ADD COLUMN IF NOT EXISTS "attached_by_id" varchar(100);
