ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "accessorial_charge_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "charge_code" varchar(10),
    ADD COLUMN IF NOT EXISTS "charge_method" varchar(20),
    ADD COLUMN IF NOT EXISTS "rate_unit" varchar(20),
    ADD COLUMN IF NOT EXISTS "rate" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rate_basis_amount" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "formula_template_name" text;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_lines_accessorial_charge" ON "invoice_lines"("accessorial_charge_id", "organization_id", "business_unit_id")
WHERE
    "accessorial_charge_id" IS NOT NULL;

--bun:split
WITH "line_ranked" AS (
    SELECT
        il."id",
        il."organization_id",
        il."business_unit_id",
        il."invoice_id",
        il."shipment_id",
        il."amount",
        ROW_NUMBER() OVER (PARTITION BY il."invoice_id", il."organization_id", il."business_unit_id", il."shipment_id" ORDER BY il."line_number") AS "rn",
        COUNT(*) OVER (PARTITION BY il."invoice_id", il."organization_id", il."business_unit_id", il."shipment_id") AS "cnt"
    FROM
        "invoice_lines" il
    WHERE
        il."type" = 'Accessorial'
        AND il."shipment_id" IS NOT NULL
        AND il."accessorial_charge_id" IS NULL
),
"freight" AS (
    SELECT
        il."invoice_id",
        il."organization_id",
        il."business_unit_id",
        il."shipment_id",
        MAX(ABS(il."amount")) AS "amount"
    FROM
        "invoice_lines" il
    WHERE
        il."type" = 'Freight'
        AND il."shipment_id" IS NOT NULL
    GROUP BY
        il."invoice_id",
        il."organization_id",
        il."business_unit_id",
        il."shipment_id"
),
"charge_ranked" AS (
    SELECT
        ac."shipment_id",
        ac."organization_id",
        ac."business_unit_id",
        ac."accessorial_charge_id",
        ac."method"::text AS "method",
        ac."amount",
        ac."unit",
        ROW_NUMBER() OVER (PARTITION BY ac."shipment_id", ac."organization_id", ac."business_unit_id" ORDER BY ac."id") AS "rn",
        COUNT(*) OVER (PARTITION BY ac."shipment_id", ac."organization_id", ac."business_unit_id") AS "cnt"
    FROM
        "additional_charges" ac
),
"matched" AS (
    SELECT
        lr."id",
        lr."organization_id",
        lr."business_unit_id",
        acc."id" AS "accessorial_charge_id",
        acc."code",
        acc."description",
        acc."rate_unit"::text AS "rate_unit",
        cr."method",
        cr."amount" AS "rate",
        f."amount" AS "freight_amount"
    FROM
        "line_ranked" lr
        JOIN "charge_ranked" cr ON cr."shipment_id" = lr."shipment_id"
            AND cr."organization_id" = lr."organization_id"
            AND cr."business_unit_id" = lr."business_unit_id"
            AND cr."rn" = lr."rn"
            AND cr."cnt" = lr."cnt"
        JOIN "accessorial_charges" acc ON acc."id" = cr."accessorial_charge_id"
            AND acc."organization_id" = cr."organization_id"
            AND acc."business_unit_id" = cr."business_unit_id"
        LEFT JOIN "freight" f ON f."invoice_id" = lr."invoice_id"
            AND f."organization_id" = lr."organization_id"
            AND f."business_unit_id" = lr."business_unit_id"
            AND f."shipment_id" = lr."shipment_id"
    WHERE
        ABS(ABS(lr."amount") - CASE cr."method"
            WHEN 'Flat' THEN cr."amount" * GREATEST(cr."unit", 1)
            WHEN 'PerUnit' THEN CASE WHEN cr."unit" < 1 THEN 0 ELSE cr."amount" * cr."unit" END
            WHEN 'Percentage' THEN COALESCE(f."amount", 0) * cr."amount" / 100
            ELSE NULL
        END) < 0.01)
UPDATE
    "invoice_lines" il
SET
    "accessorial_charge_id" = m."accessorial_charge_id",
    "charge_code" = m."code",
    "charge_method" = m."method",
    "rate_unit" = CASE WHEN m."method" = 'PerUnit' THEN m."rate_unit" ELSE NULL END,
    "rate" = m."rate",
    "rate_basis_amount" = CASE WHEN m."method" = 'Percentage' THEN m."freight_amount" ELSE NULL END,
    "description" = CASE WHEN il."description" = 'Accessorial charge'
        AND COALESCE(TRIM(m."description"), '') <> '' THEN m."description"
    ELSE il."description" END
FROM
    "matched" m
WHERE
    il."id" = m."id"
    AND il."organization_id" = m."organization_id"
    AND il."business_unit_id" = m."business_unit_id";
