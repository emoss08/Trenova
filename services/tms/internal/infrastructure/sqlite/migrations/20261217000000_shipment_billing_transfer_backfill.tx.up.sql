-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261217000000_shipment_billing_transfer_backfill.tx.up.sql
--
-- Hand-completed: SQLite has no DISTINCT ON or UPDATE ... FROM with a derived
-- ordering, so the latest invoice item is picked with correlated subqueries.

UPDATE "shipments"
SET "billing_transfer_status" = (
        SELECT bqi."status"
        FROM "billing_queue_items" AS bqi
        WHERE bqi."shipment_id" = "shipments"."id"
            AND bqi."organization_id" = "shipments"."organization_id"
            AND bqi."business_unit_id" = "shipments"."business_unit_id"
            AND bqi."bill_type" = 'Invoice'
        ORDER BY bqi."created_at" DESC, bqi."id" DESC
        LIMIT 1
    ),
    "transferred_to_billing_at" = (
        SELECT bqi."created_at"
        FROM "billing_queue_items" AS bqi
        WHERE bqi."shipment_id" = "shipments"."id"
            AND bqi."organization_id" = "shipments"."organization_id"
            AND bqi."business_unit_id" = "shipments"."business_unit_id"
            AND bqi."bill_type" = 'Invoice'
        ORDER BY bqi."created_at" DESC, bqi."id" DESC
        LIMIT 1
    )
WHERE EXISTS (
    SELECT 1
    FROM "billing_queue_items" AS bqi
    WHERE bqi."shipment_id" = "shipments"."id"
        AND bqi."organization_id" = "shipments"."organization_id"
        AND bqi."business_unit_id" = "shipments"."business_unit_id"
        AND bqi."bill_type" = 'Invoice'
);
