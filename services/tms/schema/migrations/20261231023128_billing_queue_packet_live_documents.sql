-- A deleted, superseded, moved, rejected or archived document stayed in the
-- packets that carried it, and every later save of the item's invoice details
-- was refused for it. Each packet now keeps only documents its shipment still
-- holds as current, and the turns of those documents; the document observer
-- keeps it that way from here on.
WITH "live" AS (
    SELECT "d"."id", "d"."organization_id", "d"."business_unit_id", "d"."resource_id"
    FROM "documents" AS "d"
    WHERE "d"."resource_type" = 'shipment'
        AND "d"."is_current_version"
        AND "d"."status" NOT IN ('Rejected', 'Archived')
)
UPDATE "billing_queue_items" AS "bq"
SET
    "packet_document_ids" = COALESCE((
        SELECT jsonb_agg(to_jsonb("packet"."id") ORDER BY "packet"."at")
        FROM jsonb_array_elements_text("bq"."packet_document_ids")
            WITH ORDINALITY AS "packet"("id", "at")
        WHERE EXISTS (
            SELECT 1 FROM "live"
            WHERE "live"."id" = "packet"."id"
                AND "live"."organization_id" = "bq"."organization_id"
                AND "live"."business_unit_id" = "bq"."business_unit_id"
                AND "live"."resource_id" = "bq"."shipment_id"
        )
    ), '[]'::jsonb),
    "packet_page_rotations" = COALESCE((
        SELECT jsonb_agg("turn"."value" ORDER BY "turn"."at")
        FROM jsonb_array_elements("bq"."packet_page_rotations")
            WITH ORDINALITY AS "turn"("value", "at")
        WHERE EXISTS (
            SELECT 1 FROM "live"
            WHERE "live"."id" = "turn"."value" ->> 'documentId'
                AND "live"."organization_id" = "bq"."organization_id"
                AND "live"."business_unit_id" = "bq"."business_unit_id"
                AND "live"."resource_id" = "bq"."shipment_id"
        )
    ), '[]'::jsonb)
WHERE jsonb_array_length("bq"."packet_document_ids") > 0
    AND EXISTS (
        SELECT 1
        FROM jsonb_array_elements_text("bq"."packet_document_ids") AS "carried"("id")
        WHERE NOT EXISTS (
            SELECT 1 FROM "live"
            WHERE "live"."id" = "carried"."id"
                AND "live"."organization_id" = "bq"."organization_id"
                AND "live"."business_unit_id" = "bq"."business_unit_id"
                AND "live"."resource_id" = "bq"."shipment_id"
        )
    );
