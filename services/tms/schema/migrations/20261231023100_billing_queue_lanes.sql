ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "lane" varchar(20) NOT NULL DEFAULT 'Checking',
    ADD COLUMN IF NOT EXISTS "checked_at" bigint,
    ADD COLUMN IF NOT EXISTS "po_number" varchar(100),
    ADD COLUMN IF NOT EXISTS "invoice_memo" text,
    ADD COLUMN IF NOT EXISTS "payment_term" varchar(50),
    ADD COLUMN IF NOT EXISTS "invoice_date" bigint,
    ADD COLUMN IF NOT EXISTS "packet_document_ids" jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "ck_billing_queue_items_lane" CHECK ("lane" IN ('Checking', 'NeedsYou', 'Waiting', 'Ready', 'Held', 'Approved', 'Closed'));

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "po_number" varchar(100);

UPDATE
    "billing_queue_items" AS bqi
SET
    "po_number" = NULLIF(TRIM(o."po_number"), '')
FROM
    "orders" AS o
WHERE
    o."id" = bqi."order_id"
    AND o."organization_id" = bqi."organization_id"
    AND o."business_unit_id" = bqi."business_unit_id"
    AND bqi."po_number" IS NULL;

UPDATE
    "billing_queue_items"
SET
    "lane" = CASE WHEN "status" IN ('Posted', 'Canceled') THEN
        'Closed'
    WHEN "status" = 'Approved' THEN
        'Approved'
    WHEN "status" IN ('OnHold', 'Exception', 'SentBackToOps') THEN
        'Held'
    ELSE
        'Checking'
    END;
