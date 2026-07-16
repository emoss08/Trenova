--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Credit-memo and replacement billing-queue items for grouped (order) invoices carry
-- an order id instead of a single shipment id — mirror the invoices.shipment_id change.
ALTER TABLE "billing_queue_items"
    ALTER COLUMN "shipment_id" DROP NOT NULL;

--bun:split
ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "chk_billing_queue_items_shipment_or_order" CHECK ("shipment_id" IS NOT NULL OR "order_id" IS NOT NULL) NOT VALID;

--bun:split
ALTER TABLE "billing_queue_items" VALIDATE CONSTRAINT "chk_billing_queue_items_shipment_or_order";
