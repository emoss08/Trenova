-- The open items each unapplied bank receipt most plausibly pays, ranked, with the evidence the
-- receipt itself carries: a posted customer payment it resembles (reference, amount, date), or an
-- open invoice whose number is a whole token of its reference or memo. Written only by the
-- bank receipt candidate drain (billing/jobs/bankreceiptcandidatejobs), replacing a receipt's
-- rows as one set, so a retry writes the same rows. Read by the reconciliation page's
-- suggestions, the Desk card and the board; never recomputed on a request.
CREATE TABLE IF NOT EXISTS "bank_receipt_candidates" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "bank_receipt_id" varchar(100) NOT NULL,
    "rank" integer NOT NULL,
    "candidate_kind" varchar(30) NOT NULL,
    "customer_payment_id" varchar(100),
    "invoice_id" varchar(100),
    "customer_id" varchar(100),
    "score" integer NOT NULL,
    "evidence" text[] NOT NULL DEFAULT '{}',
    "standing" varchar(20) NOT NULL,
    "open_minor" bigint NOT NULL,
    "difference_minor" bigint NOT NULL,
    "computed_at" bigint NOT NULL,
    CONSTRAINT "pk_bank_receipt_candidates" PRIMARY KEY ("organization_id", "business_unit_id", "bank_receipt_id", "rank"),
    CONSTRAINT "ck_bank_receipt_candidates_rank" CHECK ("rank" BETWEEN 1 AND 10),
    CONSTRAINT "ck_bank_receipt_candidates_kind" CHECK (
        ("candidate_kind" = 'CustomerPayment' AND "customer_payment_id" IS NOT NULL AND "invoice_id" IS NULL)
        OR ("candidate_kind" = 'Invoice' AND "invoice_id" IS NOT NULL AND "customer_payment_id" IS NULL)),
    CONSTRAINT "ck_bank_receipt_candidates_standing" CHECK (
        "standing" IN ('Exact', 'SameAmount', 'Short', 'Over')),
    CONSTRAINT "fk_bank_receipt_candidates_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_bank_receipt_candidates_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_bank_receipt_candidates_receipt" FOREIGN KEY ("bank_receipt_id", "organization_id", "business_unit_id")
        REFERENCES "bank_receipts" ("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_bank_receipt_candidates_payment" FOREIGN KEY ("customer_payment_id", "organization_id", "business_unit_id")
        REFERENCES "customer_payments" ("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_bank_receipt_candidates_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id")
        REFERENCES "invoices" ("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

-- Receipts by how their best candidate stands (exact, short, over), for the board's counts.
CREATE INDEX IF NOT EXISTS "idx_bank_receipt_candidates_best"
    ON "bank_receipt_candidates" ("organization_id", "business_unit_id", "standing")
    INCLUDE ("bank_receipt_id")
    WHERE "rank" = 1;

-- The receipts that list a changed invoice or payment, so the drain recomputes them.
CREATE INDEX IF NOT EXISTS "idx_bank_receipt_candidates_invoice"
    ON "bank_receipt_candidates" ("organization_id", "business_unit_id", "invoice_id")
    WHERE "invoice_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_bank_receipt_candidates_payment"
    ON "bank_receipt_candidates" ("organization_id", "business_unit_id", "customer_payment_id")
    WHERE "customer_payment_id" IS NOT NULL;

-- A change that can move a receipt's candidates queues its subject for the tenant's drain
-- (work_queue_items, kind bank-receipt-candidates), in the writer's transaction, whatever
-- service made the change. The drain is woken after an import or a match, and by a one-minute
-- backstop for everything else; it resolves which receipts a subject touches.
CREATE OR REPLACE FUNCTION trenova_bank_receipt_candidates_queue(
    p_organization_id varchar,
    p_business_unit_id varchar,
    p_subject_id varchar)
    RETURNS void
    LANGUAGE sql
AS $$
    INSERT INTO "work_queue_items" ("organization_id", "business_unit_id", "kind", "subject_id", "due_at")
    VALUES (p_organization_id, p_business_unit_id, 'bank-receipt-candidates', p_subject_id,
        extract(epoch FROM clock_timestamp())::bigint)
    ON CONFLICT ("organization_id", "business_unit_id", "kind", "subject_id")
    DO UPDATE SET
        "due_at" = GREATEST("work_queue_items"."due_at" + 1, EXCLUDED."due_at"),
        "updated_at" = EXCLUDED."due_at";
$$;

-- An open item queues itself only while its tenant has a receipt waiting to be applied: one
-- index probe (idx_bank_receipts_status_date) for a tenant that does not import bank receipts.
CREATE OR REPLACE FUNCTION trenova_bank_receipt_candidates_open_item()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM "bank_receipts" br
        WHERE br."organization_id" = NEW."organization_id"
            AND br."business_unit_id" = NEW."business_unit_id"
            AND br."status" IN ('Imported', 'Exception')) THEN
        PERFORM trenova_bank_receipt_candidates_queue(
            NEW."organization_id", NEW."business_unit_id", NEW."id");
    END IF;
    RETURN NULL;
END
$$;

CREATE OR REPLACE TRIGGER "trg_invoices_bank_receipt_candidates_insert"
    AFTER INSERT ON "invoices"
    FOR EACH ROW
    WHEN (NEW."status" = 'Posted')
    EXECUTE FUNCTION trenova_bank_receipt_candidates_open_item();

CREATE OR REPLACE TRIGGER "trg_invoices_bank_receipt_candidates_update"
    AFTER UPDATE ON "invoices"
    FOR EACH ROW
    WHEN (OLD."status" IS DISTINCT FROM NEW."status"
        OR OLD."number" IS DISTINCT FROM NEW."number"
        OR OLD."customer_id" IS DISTINCT FROM NEW."customer_id"
        OR OLD."bill_type" IS DISTINCT FROM NEW."bill_type"
        OR OLD."total_amount_minor" IS DISTINCT FROM NEW."total_amount_minor"
        OR OLD."applied_amount_minor" IS DISTINCT FROM NEW."applied_amount_minor")
    EXECUTE FUNCTION trenova_bank_receipt_candidates_open_item();

CREATE OR REPLACE TRIGGER "trg_customer_payments_bank_receipt_candidates_insert"
    AFTER INSERT ON "customer_payments"
    FOR EACH ROW
    WHEN (NEW."status" = 'Posted')
    EXECUTE FUNCTION trenova_bank_receipt_candidates_open_item();

CREATE OR REPLACE TRIGGER "trg_customer_payments_bank_receipt_candidates_update"
    AFTER UPDATE ON "customer_payments"
    FOR EACH ROW
    WHEN (OLD."status" IS DISTINCT FROM NEW."status"
        OR OLD."customer_id" IS DISTINCT FROM NEW."customer_id"
        OR OLD."reference_number" IS DISTINCT FROM NEW."reference_number"
        OR OLD."amount_minor" IS DISTINCT FROM NEW."amount_minor"
        OR OLD."payment_date" IS DISTINCT FROM NEW."payment_date")
    EXECUTE FUNCTION trenova_bank_receipt_candidates_open_item();

-- A receipt queues itself when it arrives or changes; a match or its release also queues the
-- payment on both sides, since every other receipt that listed it must be recomputed.
CREATE OR REPLACE FUNCTION trenova_bank_receipt_candidates_receipt()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM trenova_bank_receipt_candidates_queue(
        NEW."organization_id", NEW."business_unit_id", NEW."id");
    IF TG_OP = 'UPDATE'
        AND OLD."matched_customer_payment_id" IS DISTINCT FROM NEW."matched_customer_payment_id" THEN
        IF OLD."matched_customer_payment_id" IS NOT NULL THEN
            PERFORM trenova_bank_receipt_candidates_queue(
                NEW."organization_id", NEW."business_unit_id", OLD."matched_customer_payment_id");
        END IF;
        IF NEW."matched_customer_payment_id" IS NOT NULL THEN
            PERFORM trenova_bank_receipt_candidates_queue(
                NEW."organization_id", NEW."business_unit_id", NEW."matched_customer_payment_id");
        END IF;
    END IF;
    RETURN NULL;
END
$$;

CREATE OR REPLACE TRIGGER "trg_bank_receipts_candidates_insert"
    AFTER INSERT ON "bank_receipts"
    FOR EACH ROW
    EXECUTE FUNCTION trenova_bank_receipt_candidates_receipt();

CREATE OR REPLACE TRIGGER "trg_bank_receipts_candidates_update"
    AFTER UPDATE ON "bank_receipts"
    FOR EACH ROW
    WHEN (OLD."status" IS DISTINCT FROM NEW."status"
        OR OLD."amount_minor" IS DISTINCT FROM NEW."amount_minor"
        OR OLD."reference_number" IS DISTINCT FROM NEW."reference_number"
        OR OLD."memo" IS DISTINCT FROM NEW."memo"
        OR OLD."receipt_date" IS DISTINCT FROM NEW."receipt_date"
        OR OLD."matched_customer_payment_id" IS DISTINCT FROM NEW."matched_customer_payment_id")
    EXECUTE FUNCTION trenova_bank_receipt_candidates_receipt();

-- Backfill: every receipt waiting to be applied is queued; the worker's one-minute backstop
-- wakes each tenant's drain, which computes them off the request path.
INSERT INTO "work_queue_items" ("organization_id", "business_unit_id", "kind", "subject_id", "due_at")
SELECT br."organization_id", br."business_unit_id", 'bank-receipt-candidates', br."id",
    extract(epoch FROM now())::bigint
FROM "bank_receipts" br
WHERE br."status" IN ('Imported', 'Exception')
ON CONFLICT ("organization_id", "business_unit_id", "kind", "subject_id") DO NOTHING;
