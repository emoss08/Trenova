-- A biller turns a sideways page of a packet document from the packet preview.
-- The item keeps each turn by document and page, so it follows the document
-- when the packet is reordered; approving copies a document's turns onto its
-- invoice attachment, and every packet built from the invoice applies them.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "packet_page_rotations" jsonb NOT NULL DEFAULT '[]';

ALTER TABLE "invoice_attachments"
    ADD COLUMN IF NOT EXISTS "page_rotations" jsonb NOT NULL DEFAULT '[]';
