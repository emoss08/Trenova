CREATE OR REPLACE FUNCTION pg_temp.accessorial_code(raw text)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
    AS $$
    SELECT
        left(rpad(CASE WHEN cleaned ~ '^[A-Z]' THEN
                    cleaned
                ELSE
                    'X' || cleaned
                END, 3, 'X'), 10)
    FROM (
        SELECT
            regexp_replace(upper(btrim(raw)), '[^A-Z0-9_]', '_', 'g') AS cleaned) normalized
$$;

--bun:split
UPDATE
    "accessorial_charges"
SET
    "code" = pg_temp.accessorial_code("code")
WHERE
    "code" IS DISTINCT FROM pg_temp.accessorial_code("code");

--bun:split
DO $$
DECLARE
    duplicate record;
    suffix integer;
    candidate text;
BEGIN
    FOR duplicate IN
    SELECT
        ranked."id",
        ranked."organization_id",
        ranked."business_unit_id",
        ranked."code"
    FROM (
        SELECT
            acc."id",
            acc."organization_id",
            acc."business_unit_id",
            acc."code",
            row_number() OVER (PARTITION BY acc."organization_id", acc."business_unit_id", acc."code" ORDER BY acc."created_at", acc."id") AS "position"
        FROM
            "accessorial_charges" acc) ranked
    WHERE
        ranked."position" > 1
    ORDER BY
        ranked."organization_id",
        ranked."business_unit_id",
        ranked."code",
        ranked."id" LOOP
            suffix := 2;
            LOOP
                candidate := left(duplicate."code", 10 - length(suffix::text)) || suffix::text;
                EXIT
                WHEN NOT EXISTS (
                    SELECT
                        1
                    FROM
                        "accessorial_charges" acc
                    WHERE
                        acc."organization_id" = duplicate."organization_id"
                        AND acc."business_unit_id" = duplicate."business_unit_id"
                        AND acc."code" = candidate);
                suffix := suffix + 1;
            END LOOP;
            UPDATE
                "accessorial_charges"
            SET
                "code" = candidate
            WHERE
                "id" = duplicate."id"
                AND "organization_id" = duplicate."organization_id"
                AND "business_unit_id" = duplicate."business_unit_id";
        END LOOP;
END
$$;

--bun:split
UPDATE
    "invoice_lines" il
SET
    "charge_code" = acc."code"
FROM
    "accessorial_charges" acc
WHERE
    acc."id" = il."accessorial_charge_id"
    AND acc."organization_id" = il."organization_id"
    AND acc."business_unit_id" = il."business_unit_id"
    AND il."charge_code" IS DISTINCT FROM acc."code";

--bun:split
ALTER TABLE "accessorial_charges"
    ADD CONSTRAINT "chk_accessorial_charges_code_format" CHECK ("code" ~ '^[A-Z][A-Z0-9_]{2,9}$');
