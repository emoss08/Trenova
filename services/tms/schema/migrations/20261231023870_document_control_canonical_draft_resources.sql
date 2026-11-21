-- Each record kind a shipment draft may come from is stored by its canonical name, once, in the
-- order it was first chosen: any letter case is lowered, and the shipment aliases (an import, a
-- forwarded tender, a conversation attachment) become "shipment". A value that names no kind is
-- dropped. An organization left with drafting on and no kind to draft from drafted nothing, so
-- drafting is switched off for it, which is what it already did.
WITH canonical AS (
    SELECT
        dc."id",
        dc."organization_id",
        dc."business_unit_id",
        ARRAY(
            SELECT named."resource"
            FROM (
                SELECT
                    CASE lower(btrim(entry."value"))
                        WHEN 'shipment_import' THEN 'shipment'
                        WHEN 'inbound_message' THEN 'shipment'
                        WHEN 'assistant_thread' THEN 'shipment'
                        ELSE lower(btrim(entry."value"))
                    END AS "resource",
                    entry."position"
                FROM unnest(dc."shipment_draft_allowed_resources")
                    WITH ORDINALITY AS entry("value", "position")
            ) named
            WHERE named."resource" IN ('shipment', 'trailer', 'tractor', 'worker')
            GROUP BY named."resource"
            ORDER BY min(named."position")
        )::VARCHAR(100)[] AS "resources"
    FROM "document_controls" dc
)
UPDATE "document_controls" dc
SET "shipment_draft_allowed_resources" = canonical."resources",
    "enable_shipment_draft_extraction" = dc."enable_shipment_draft_extraction"
        AND cardinality(canonical."resources") > 0,
    "version" = dc."version" + 1
FROM canonical
WHERE dc."id" = canonical."id"
    AND dc."organization_id" = canonical."organization_id"
    AND dc."business_unit_id" = canonical."business_unit_id"
    AND (
        dc."shipment_draft_allowed_resources" IS DISTINCT FROM canonical."resources"
        OR (dc."enable_shipment_draft_extraction" AND cardinality(canonical."resources") = 0)
    );

ALTER TABLE "document_controls"
    ADD CONSTRAINT "ck_document_controls_shipment_draft_allowed_resources"
    CHECK ("shipment_draft_allowed_resources" <@ ARRAY['shipment', 'trailer', 'tractor', 'worker']::VARCHAR(100)[]);
