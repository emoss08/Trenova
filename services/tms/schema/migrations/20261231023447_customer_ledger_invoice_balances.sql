-- What each of a customer's invoices still carries on the customer's ledger: the sum of the entries
-- tied to the invoice (its own posting, and every payment, credit, short pay and factoring entry
-- that names it), kept by statement triggers on the ledger in the transaction that writes it.
-- A customer's credit exposure leaves out the invoices sold to a non-recourse factor, and whether
-- an invoice is sold that way is decided now, from the invoice, its agreement, the factor and its
-- charge-backs, so it cannot be kept here: the exposure is the customer's monthly balances less
-- the rows of this table whose invoice is sold without recourse today. A settled invoice nets to
-- zero and drops out of the partial index, so that read touches the invoices still carrying a
-- balance, never the customer's whole history.
CREATE TABLE IF NOT EXISTS "customer_ledger_invoice_balances"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "amount_minor" bigint NOT NULL DEFAULT 0,
    CONSTRAINT "pk_customer_ledger_invoice_balances" PRIMARY KEY ("organization_id", "business_unit_id", "customer_id", "invoice_id"),
    CONSTRAINT "fk_customer_ledger_invoice_balances_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "customer_ledger_invoice_balances" IS 'The sum of each customer''s ledger entries per invoice they name, kept by triggers on customer_ledger_entries';

COMMENT ON COLUMN "customer_ledger_invoice_balances"."invoice_id" IS 'The entry''s related invoice, or the entry''s own source when it is an invoice';

COMMENT ON COLUMN "customer_ledger_invoice_balances"."amount_minor" IS 'The sum of amount_minor over the customer''s entries for the invoice';

CREATE INDEX IF NOT EXISTS "idx_customer_ledger_invoice_balances_carrying" ON "customer_ledger_invoice_balances"("organization_id", "business_unit_id", "customer_id") INCLUDE ("invoice_id", "amount_minor")
WHERE
    "amount_minor" <> 0;

CREATE OR REPLACE FUNCTION customer_ledger_invoice_id(related_invoice_id varchar, source_object_type varchar, source_object_id varchar) RETURNS varchar
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $fn$
    SELECT COALESCE(related_invoice_id, CASE WHEN source_object_type = 'Invoice' THEN source_object_id END)
$fn$;

CREATE OR REPLACE FUNCTION customer_ledger_invoice_balances_apply() RETURNS trigger
LANGUAGE plpgsql AS $fn$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO "customer_ledger_invoice_balances" AS t ("organization_id", "business_unit_id", "customer_id", "invoice_id", "amount_minor")
        SELECT n."organization_id", n."business_unit_id", n."customer_id", n."invoice_id", sum(n."amount_minor")
          FROM (
                SELECT "organization_id", "business_unit_id", "customer_id", "amount_minor",
                       customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id") AS "invoice_id"
                  FROM customer_ledger_new
               ) n
         WHERE n."invoice_id" IS NOT NULL
         GROUP BY 1, 2, 3, 4
         ORDER BY 1, 2, 3, 4
        ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "invoice_id")
        DO UPDATE SET "amount_minor" = t."amount_minor" + EXCLUDED."amount_minor";
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE "customer_ledger_invoice_balances" t
           SET "amount_minor" = t."amount_minor" - o."amount_minor"
          FROM (
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id") AS "invoice_id",
                       sum("amount_minor") AS "amount_minor"
                  FROM customer_ledger_old
                 GROUP BY 1, 2, 3, 4
               ) o
         WHERE t."organization_id" = o."organization_id"
           AND t."business_unit_id" = o."business_unit_id"
           AND t."customer_id" = o."customer_id"
           AND t."invoice_id" = o."invoice_id";
    ELSE
        INSERT INTO "customer_ledger_invoice_balances" AS t ("organization_id", "business_unit_id", "customer_id", "invoice_id", "amount_minor")
        SELECT c."organization_id", c."business_unit_id", c."customer_id", c."invoice_id", sum(c."delta")
          FROM (
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id") AS "invoice_id",
                       "amount_minor" AS "delta"
                  FROM customer_ledger_new
                UNION ALL
                SELECT "organization_id", "business_unit_id", "customer_id",
                       customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id"),
                       -"amount_minor"
                  FROM customer_ledger_old
               ) c
         WHERE c."invoice_id" IS NOT NULL
         GROUP BY 1, 2, 3, 4
        HAVING sum(c."delta") <> 0
         ORDER BY 1, 2, 3, 4
        ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "invoice_id")
        DO UPDATE SET "amount_minor" = t."amount_minor" + EXCLUDED."amount_minor";
    END IF;
    RETURN NULL;
END
$fn$;

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_invoice_insert"
    AFTER INSERT ON "customer_ledger_entries"
    REFERENCING NEW TABLE AS customer_ledger_new
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_invoice_balances_apply();

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_invoice_update"
    AFTER UPDATE ON "customer_ledger_entries"
    REFERENCING OLD TABLE AS customer_ledger_old NEW TABLE AS customer_ledger_new
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_invoice_balances_apply();

CREATE OR REPLACE TRIGGER "trg_customer_ledger_entries_invoice_delete"
    AFTER DELETE ON "customer_ledger_entries"
    REFERENCING OLD TABLE AS customer_ledger_old
    FOR EACH STATEMENT EXECUTE FUNCTION customer_ledger_invoice_balances_apply();

INSERT INTO "customer_ledger_invoice_balances" ("organization_id", "business_unit_id", "customer_id", "invoice_id", "amount_minor")
SELECT "organization_id", "business_unit_id", "customer_id",
       customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id"), sum("amount_minor")
  FROM "customer_ledger_entries"
 WHERE customer_ledger_invoice_id("related_invoice_id", "source_object_type", "source_object_id") IS NOT NULL
 GROUP BY 1, 2, 3, 4
ON CONFLICT ("organization_id", "business_unit_id", "customer_id", "invoice_id")
DO UPDATE SET "amount_minor" = EXCLUDED."amount_minor";
