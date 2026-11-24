-- Fuel bought and miles run by each tractor on each day of its organization's calendar, kept so
-- cost per mile reads a month of days by tractor instead of every purchase, mileage entry and
-- completed move of the month. fuel_tractor_daily_refresh(days) rebuilds the trailing days of
-- every organization in one statement, idempotently: a short run every few minutes keeps today
-- and yesterday current, a nightly run re-reads the trailing two months for back-dated edits.
-- Miles are what the IFTA return counts a tractor for: completed moves' fuel tax distance on
-- the day the move finished, unless jurisdiction mileage entries were recorded for the move,
-- plus those entries on the day traveled.
CREATE TABLE IF NOT EXISTS "fuel_tractor_daily" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "day_at" bigint NOT NULL,
    "tractor_id" varchar(100) NOT NULL,
    "purchases" integer NOT NULL DEFAULT 0,
    "gallons" numeric(14, 3) NOT NULL DEFAULT 0,
    "amount_minor" bigint NOT NULL DEFAULT 0,
    "currency_code" varchar(3),
    "currencies" integer NOT NULL DEFAULT 0,
    "miles" numeric(14, 2) NOT NULL DEFAULT 0,
    "refreshed_at" bigint NOT NULL,
    CONSTRAINT "pk_fuel_tractor_daily" PRIMARY KEY ("organization_id", "business_unit_id", "day_at", "tractor_id"),
    CONSTRAINT "fk_fuel_tractor_daily_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_tractor_daily_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units" ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_fuel_tractor_daily_tractor"
    ON "fuel_tractor_daily" ("organization_id", "business_unit_id", "tractor_id", "day_at");

CREATE OR REPLACE FUNCTION fuel_tractor_daily_refresh(p_days integer)
    RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    written bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('fuel_tractor_daily_refresh'));

    WITH fuel_tractor_daily_zone AS (
        SELECT
            o."id" AS organization_id,
            z.tz,
            extract(epoch FROM (((now() AT TIME ZONE z.tz)::date - p_days)::timestamp AT TIME ZONE z.tz))::bigint AS first_at
        FROM
            "organizations" AS o
            CROSS JOIN LATERAL (SELECT coalesce(nullif(o."timezone", ''), 'UTC') AS tz) AS z
    )
    DELETE FROM "fuel_tractor_daily" AS d
    USING fuel_tractor_daily_zone AS z
    WHERE d."organization_id" = z.organization_id
        AND d."day_at" >= z.first_at;

    WITH fuel_tractor_daily_zone AS (
        SELECT
            o."id" AS organization_id,
            z.tz,
            extract(epoch FROM (((now() AT TIME ZONE z.tz)::date - p_days)::timestamp AT TIME ZONE z.tz))::bigint AS first_at
        FROM
            "organizations" AS o
            CROSS JOIN LATERAL (SELECT coalesce(nullif(o."timezone", ''), 'UTC') AS tz) AS z
    )
    INSERT INTO "fuel_tractor_daily" (
        "organization_id", "business_unit_id", "day_at", "tractor_id", "purchases", "gallons",
        "amount_minor", "currency_code", "currencies", "miles", "refreshed_at")
    SELECT
        s.organization_id,
        s.business_unit_id,
        s.day_at,
        s.tractor_id,
        sum(s.purchases),
        sum(s.gallons),
        sum(s.amount_minor),
        min(s.currency_code),
        count(DISTINCT s.currency_code),
        sum(s.miles),
        extract(epoch FROM now())::bigint
    FROM (
        SELECT
            p."organization_id",
            p."business_unit_id",
            p."tractor_id",
            extract(epoch FROM ((to_timestamp(p."purchased_at") AT TIME ZONE z.tz)::date::timestamp AT TIME ZONE z.tz))::bigint AS day_at,
            1 AS purchases,
            p."gallons",
            p."total_amount_minor" AS amount_minor,
            p."currency_code"::text AS currency_code,
            0::numeric AS miles
        FROM
            "fuel_purchases" AS p
            JOIN fuel_tractor_daily_zone AS z ON z.organization_id = p."organization_id"
        WHERE
            p."purchased_at" >= z.first_at
        UNION ALL
        SELECT
            e."organization_id",
            e."business_unit_id",
            e."tractor_id",
            extract(epoch FROM ((to_timestamp(e."traveled_at") AT TIME ZONE z.tz)::date::timestamp AT TIME ZONE z.tz))::bigint,
            0,
            0,
            0,
            NULL,
            e."miles"
        FROM
            "ifta_jurisdiction_mileage_entries" AS e
            JOIN fuel_tractor_daily_zone AS z ON z.organization_id = e."organization_id"
        WHERE
            e."traveled_at" >= z.first_at
        UNION ALL
        SELECT
            m.organization_id,
            m.business_unit_id,
            m.tractor_id,
            extract(epoch FROM ((to_timestamp(m.done_at) AT TIME ZONE m.tz)::date::timestamp AT TIME ZONE m.tz))::bigint,
            0,
            0,
            0,
            NULL,
            m.miles
        FROM (
            SELECT
                sm."organization_id",
                sm."business_unit_id",
                a."tractor_id",
                z.tz,
                z.first_at,
                round((CASE WHEN sm."distance_units" = 'Kilometers'
                    THEN coalesce(sm."fuel_distance", sm."distance") / 1.609344
                    ELSE coalesce(sm."fuel_distance", sm."distance")
                END)::numeric, 2) AS miles,
                coalesce(max(coalesce(st."actual_departure", st."actual_arrival")), sm."updated_at") AS done_at
            FROM
                "shipment_moves" AS sm
                JOIN fuel_tractor_daily_zone AS z ON z.organization_id = sm."organization_id"
                JOIN "assignments" AS a ON a."shipment_move_id" = sm."id"
                    AND a."organization_id" = sm."organization_id"
                    AND a."business_unit_id" = sm."business_unit_id"
                    AND a."archived_at" IS NULL
                    AND a."tractor_id" IS NOT NULL
                LEFT JOIN "stops" AS st ON st."shipment_move_id" = sm."id"
                    AND st."organization_id" = sm."organization_id"
                    AND st."business_unit_id" = sm."business_unit_id"
            WHERE
                sm."status" = 'Completed'
                AND sm."updated_at" >= z.first_at
                AND NOT EXISTS (
                    SELECT 1 FROM "ifta_jurisdiction_mileage_entries" AS me
                    WHERE me."organization_id" = sm."organization_id"
                        AND me."business_unit_id" = sm."business_unit_id"
                        AND me."shipment_move_id" = sm."id")
            GROUP BY
                sm."organization_id", sm."business_unit_id", sm."id", a."tractor_id", z.tz,
                z.first_at, sm."distance", sm."fuel_distance", sm."distance_units", sm."updated_at"
        ) AS m
        WHERE
            m.done_at >= m.first_at
            AND m.miles IS NOT NULL
    ) AS s
    GROUP BY
        s.organization_id, s.business_unit_id, s.day_at, s.tractor_id;

    GET DIAGNOSTICS written = ROW_COUNT;
    RETURN written;
END;
$$;

SELECT fuel_tractor_daily_refresh(400);
