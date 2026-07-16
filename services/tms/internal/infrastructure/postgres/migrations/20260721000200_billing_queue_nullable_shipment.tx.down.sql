--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "chk_billing_queue_items_shipment_or_order";

--bun:split
DELETE FROM "billing_queue_items"
WHERE "shipment_id" IS NULL;

--bun:split
ALTER TABLE "billing_queue_items"
    ALTER COLUMN "shipment_id" SET NOT NULL;
