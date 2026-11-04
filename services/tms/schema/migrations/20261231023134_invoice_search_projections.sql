-- What the command palette's search reads of an invoice: its number and the references a
-- person types (PRO, BOL, PO, order), who it bills, where it stands and what it is for. GTC
-- syncs this table into the "invoices" Meilisearch index, as it syncs documents' projection, so
-- the index holds only these columns and never an invoice's email snapshots, remittance text,
-- memo or address. A trigger keeps each row equal to its invoice on every write that changes a
-- column it holds (posting, payments, credits, disputes, voids, imports, seeds), and on no other,
-- so GTC is not sent an invoice's every touch; existing invoices are projected here. Days past due is never stored: it changes every day for every open invoice, and is read
-- from the due date and the balance due.
CREATE TABLE IF NOT EXISTS "invoice_search_projections" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "customer_id" VARCHAR(100),
    "number" VARCHAR(100) NOT NULL,
    "bill_type" VARCHAR(50) NOT NULL,
    "status" VARCHAR(50) NOT NULL,
    "settlement_status" VARCHAR(50),
    "dispute_status" VARCHAR(50),
    "bill_to_name" VARCHAR(255),
    "bill_to_code" VARCHAR(100),
    "shipment_pro_number" VARCHAR(100),
    "shipment_bol" VARCHAR(100),
    "order_number" VARCHAR(100),
    "po_number" VARCHAR(100),
    "currency_code" VARCHAR(10),
    "total_amount_minor" BIGINT NOT NULL DEFAULT 0,
    "balance_due_minor" BIGINT NOT NULL DEFAULT 0,
    "invoice_date" BIGINT,
    "due_date" BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_invoice_search_projections" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_invoice_search_projections_invoice" FOREIGN KEY ("id", "organization_id", "business_unit_id") REFERENCES "invoices" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_search_projections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_search_projections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "invoice_search_projections" IS 'The columns of an invoice the global search indexes, kept equal to the invoice by trigger and synced into Meilisearch by GTC';

CREATE OR REPLACE FUNCTION "invoices_search_projection"()
    RETURNS TRIGGER
    AS $$
BEGIN
    INSERT INTO "invoice_search_projections" AS p (
        "id", "organization_id", "business_unit_id", "customer_id", "number", "bill_type",
        "status", "settlement_status", "dispute_status", "bill_to_name", "bill_to_code",
        "shipment_pro_number", "shipment_bol", "order_number", "po_number", "currency_code",
        "total_amount_minor", "balance_due_minor", "invoice_date", "due_date", "updated_at")
    VALUES (
        NEW."id", NEW."organization_id", NEW."business_unit_id", NEW."customer_id", NEW."number",
        NEW."bill_type", NEW."status", NEW."settlement_status", NEW."dispute_status",
        NEW."bill_to_name", NEW."bill_to_code", NEW."shipment_pro_number", NEW."shipment_bol",
        NEW."order_number", NEW."po_number", NEW."currency_code",
        COALESCE(NEW."total_amount_minor", 0), COALESCE(NEW."balance_due_minor", 0),
        NEW."invoice_date", NEW."due_date", NEW."updated_at")
    ON CONFLICT ("id", "organization_id", "business_unit_id") DO UPDATE
    SET "customer_id" = EXCLUDED."customer_id",
        "number" = EXCLUDED."number",
        "bill_type" = EXCLUDED."bill_type",
        "status" = EXCLUDED."status",
        "settlement_status" = EXCLUDED."settlement_status",
        "dispute_status" = EXCLUDED."dispute_status",
        "bill_to_name" = EXCLUDED."bill_to_name",
        "bill_to_code" = EXCLUDED."bill_to_code",
        "shipment_pro_number" = EXCLUDED."shipment_pro_number",
        "shipment_bol" = EXCLUDED."shipment_bol",
        "order_number" = EXCLUDED."order_number",
        "po_number" = EXCLUDED."po_number",
        "currency_code" = EXCLUDED."currency_code",
        "total_amount_minor" = EXCLUDED."total_amount_minor",
        "balance_due_minor" = EXCLUDED."balance_due_minor",
        "invoice_date" = EXCLUDED."invoice_date",
        "due_date" = EXCLUDED."due_date",
        "updated_at" = EXCLUDED."updated_at"
    WHERE (p."customer_id", p."number", p."bill_type", p."status", p."settlement_status",
            p."dispute_status", p."bill_to_name", p."bill_to_code", p."shipment_pro_number",
            p."shipment_bol", p."order_number", p."po_number", p."currency_code",
            p."total_amount_minor", p."balance_due_minor", p."invoice_date", p."due_date")
        IS DISTINCT FROM (EXCLUDED."customer_id", EXCLUDED."number", EXCLUDED."bill_type",
            EXCLUDED."status", EXCLUDED."settlement_status", EXCLUDED."dispute_status",
            EXCLUDED."bill_to_name", EXCLUDED."bill_to_code", EXCLUDED."shipment_pro_number",
            EXCLUDED."shipment_bol", EXCLUDED."order_number", EXCLUDED."po_number",
            EXCLUDED."currency_code", EXCLUDED."total_amount_minor", EXCLUDED."balance_due_minor",
            EXCLUDED."invoice_date", EXCLUDED."due_date");
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_invoices_search_projection" ON "invoices";

-- Fired only by an update naming a column the projection holds, so the many writes that touch
-- none of them (send status, EDI, PDF, notes) never run it. A payment names only
-- "applied_amount_minor", which the generated "balance_due_minor" is computed from, and Postgres
-- fires a trigger listing a generated column when a column it depends on is updated. An update
-- that names a column and leaves it unchanged is caught by the IS DISTINCT FROM guard above.
CREATE TRIGGER "trg_invoices_search_projection"
    AFTER INSERT OR UPDATE OF "customer_id", "number", "bill_type", "status", "settlement_status",
        "dispute_status", "bill_to_name", "bill_to_code", "shipment_pro_number", "shipment_bol",
        "order_number", "po_number", "currency_code", "total_amount_minor", "balance_due_minor",
        "invoice_date", "due_date" ON "invoices"
    FOR EACH ROW
    EXECUTE FUNCTION "invoices_search_projection"();

INSERT INTO "invoice_search_projections" (
    "id", "organization_id", "business_unit_id", "customer_id", "number", "bill_type", "status",
    "settlement_status", "dispute_status", "bill_to_name", "bill_to_code", "shipment_pro_number",
    "shipment_bol", "order_number", "po_number", "currency_code", "total_amount_minor",
    "balance_due_minor", "invoice_date", "due_date", "updated_at")
SELECT
    i."id", i."organization_id", i."business_unit_id", i."customer_id", i."number", i."bill_type",
    i."status", i."settlement_status", i."dispute_status", i."bill_to_name", i."bill_to_code",
    i."shipment_pro_number", i."shipment_bol", i."order_number", i."po_number", i."currency_code",
    COALESCE(i."total_amount_minor", 0), COALESCE(i."balance_due_minor", 0), i."invoice_date",
    i."due_date", i."updated_at"
FROM "invoices" AS i
ON CONFLICT ("id", "organization_id", "business_unit_id") DO NOTHING;
