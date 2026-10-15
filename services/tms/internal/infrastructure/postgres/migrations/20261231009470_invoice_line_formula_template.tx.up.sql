ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100);

--bun:split
-- A freight line already names the template it was rated with. Its id is
-- taken from the shipment's rating when that rating names the same template,
-- else from the template the shipment is assigned when that one still has the
-- line's name. A template since deleted, or a name that matches neither, is
-- left without an id rather than guessed.
WITH "resolved" AS (
    SELECT
        il."id",
        il."organization_id",
        il."business_unit_id",
        COALESCE("rated"."id", "assigned"."id") AS "formula_template_id"
    FROM
        "invoice_lines" il
        JOIN "shipments" s ON s."id" = il."shipment_id"
            AND s."organization_id" = il."organization_id"
            AND s."business_unit_id" = il."business_unit_id"
        LEFT JOIN "formula_templates" "rated" ON "rated"."id" = NULLIF(TRIM(s."rating_detail" ->> 'formulaTemplateId'), '')
            AND "rated"."organization_id" = il."organization_id"
            AND "rated"."business_unit_id" = il."business_unit_id"
            AND TRIM(s."rating_detail" ->> 'formulaTemplateName') = il."formula_template_name"
        LEFT JOIN "formula_templates" "assigned" ON "assigned"."id" = s."formula_template_id"
            AND "assigned"."organization_id" = il."organization_id"
            AND "assigned"."business_unit_id" = il."business_unit_id"
            AND TRIM("assigned"."name") = il."formula_template_name"
    WHERE
        il."type" = 'Freight'
        AND il."formula_template_id" IS NULL
        AND COALESCE(TRIM(il."formula_template_name"), '') <> '')
UPDATE
    "invoice_lines" il
SET
    "formula_template_id" = r."formula_template_id"
FROM
    "resolved" r
WHERE
    il."id" = r."id"
    AND il."organization_id" = r."organization_id"
    AND il."business_unit_id" = r."business_unit_id"
    AND r."formula_template_id" IS NOT NULL;
