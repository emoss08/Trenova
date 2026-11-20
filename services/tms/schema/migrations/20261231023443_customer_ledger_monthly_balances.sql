-- Each customer's ledger added up by UTC calendar month, kept by statement triggers on the ledger
-- in the transaction that writes it, so a statement's opening balance reads one row per month of
-- the customer's history and the entries of the month it starts in, never every entry the customer
-- ever had. It is exact, not refreshed later: a statement drawn right after a payment posts must
-- already count it. Statement triggers see every writer: a posting, a bulk insert, a cascade from
-- a deleted customer and a seed's backdating update.
CREATE TABLE IF NOT EXISTS "customer_ledger_monthly_balances"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "month_start" bigint NOT NULL,
    "amount_minor" bigint NOT NULL DEFAULT 0,
    CONSTRAINT "pk_customer_ledger_monthly_balances" PRIMARY KEY ("organization_id", "business_unit_id", "customer_id", "month_start"),
    CONSTRAINT "fk_customer_ledger_monthly_balances_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "customer_ledger_monthly_balances" IS 'The sum of each customer''s ledger entries by UTC calendar month, kept by triggers on customer_ledger_entries';

COMMENT ON COLUMN "customer_ledger_monthly_balances"."month_start" IS 'The first second of the UTC calendar month, in Unix seconds';

COMMENT ON COLUMN "customer_ledger_monthly_balances"."amount_minor" IS 'The sum of amount_minor over the customer''s entries dated in the month';

CREATE OR REPLACE FUNCTION customer_ledger_month_start(epoch_seconds bigint) RETURNS bigint
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $fn$
    SELECT extract(epoch FROM date_trunc('month', to_timestamp(epoch_seconds) AT TIME ZONE 'UTC'))::bigint
$fn$;

CREATE OR REPLACE FUNCTION customer_ledger_monthly_balances_apply() RETURNS trigger
LANGUAGE plpgsql AS $fn$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO "customer_ledger_monthly_balances" AS t ("organization_id", "business_unit_id", "customer_id", "month_start", "amount_minor")
        SELECT n."organization_id", n."business_unit_id", n."customer_id",
               customer_ledger_month_start(n."transaction_date"), sum(n."amount_minor")
          FROM customer_ledger_new n
         GROUP BY 1, 2, 3, 4
         ORDER BY 1, 2, 3, 4
        ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "month_start")
        DO UPDATE SET "amount_minor" = t."amount_minor" + EXCLUDED."amount_minor";
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE "customer_ledger_monthly_balances" t
           SET "amount_minor" = t."amount_minor" - o."amount_minor"
          FROM (
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_month_start("transaction_date") AS "month_start",
                       sum("amount_minor") AS "amount_minor"
                  FROM customer_ledger_old
                 GROUP BY 1, 2, 3, 4
               ) o
         WHERE t."organization_id" = o."organization_id"
           AND t."business_unit_id" = o."business_unit_id"
           AND t."customer_id" = o."customer_id"
           AND t."month_start" = o."month_start";
    ELSE
        INSERT INTO "customer_ledger_monthly_balances" AS t ("organization_id", "business_unit_id", "customer_id", "month_start", "amount_minor")
        SELECT c."organization_id", c."business_unit_id", c."customer_id", c."month_start", sum(c."delta")
          FROM (
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_month_start("transaction_date") AS "month_start",
                       "amount_minor" AS "delta"
                  FROM customer_ledger_new
                UNION ALL
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_month_start("transaction_date"),
                       -"amount_minor"
                  FROM customer_ledger_old
               ) c
         GROUP BY 1, 2, 3, 4
        HAVING sum(c."delta") <> 0
         ORDER BY 1, 2, 3, 4
        ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "month_start")
        DO UPDATE SET "amount_minor" = t."amount_minor" + EXCLUDED."amount_minor";
    END IF;
    RETURN NULL;
END
$fn$;

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_monthly_insert"
    AFTER INSERT ON "customer_ledger_entries"
    REFERENCING NEW TABLE AS customer_ledger_new
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_monthly_balances_apply();

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_monthly_update"
    AFTER UPDATE ON "customer_ledger_entries"
    REFERENCING OLD TABLE AS customer_ledger_old NEW TABLE AS customer_ledger_new
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_monthly_balances_apply();

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_monthly_delete"
    AFTER DELETE ON "customer_ledger_entries"
    REFERENCING OLD TABLE AS customer_ledger_old
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_monthly_balances_apply();

INSERT INTO "customer_ledger_monthly_balances" ("organization_id", "business_unit_id", "customer_id", "month_start", "amount_minor")
SELECT "organization_id", "business_unit_id", "customer_id",
       customer_ledger_month_start("transaction_date"), sum("amount_minor")
  FROM "customer_ledger_entries"
 GROUP BY 1, 2, 3, 4
ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "month_start")
DO UPDATE SET "amount_minor" = EXCLUDED."amount_minor";
