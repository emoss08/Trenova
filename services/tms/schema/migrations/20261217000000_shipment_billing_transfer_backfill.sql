--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- A shipment's billing_transfer_status mirrors its latest invoice billing-queue item.
-- Before the billing queue repository became its only writer, assigning a biller,
-- posting an invoice and invoicing directly all changed the item without the
-- shipment, so existing rows are re-derived from the items once here.
UPDATE "shipments" AS sp
SET "billing_transfer_status" = latest."status"::text,
    "transferred_to_billing_at" = latest."created_at"
FROM (
    SELECT DISTINCT ON (bqi."shipment_id", bqi."organization_id", bqi."business_unit_id")
        bqi."shipment_id",
        bqi."organization_id",
        bqi."business_unit_id",
        bqi."status",
        bqi."created_at"
    FROM "billing_queue_items" AS bqi
    WHERE bqi."bill_type" = 'Invoice'
    ORDER BY
        bqi."shipment_id",
        bqi."organization_id",
        bqi."business_unit_id",
        bqi."created_at" DESC,
        bqi."id" DESC
) AS latest
WHERE sp."id" = latest."shipment_id"
    AND sp."organization_id" = latest."organization_id"
    AND sp."business_unit_id" = latest."business_unit_id";
