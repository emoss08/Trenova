-- Collections: who is chasing a customer's overdue receivables, when they
-- next follow up, and what was said and promised along the way.
--
-- collection_accounts holds one row per customer being worked: its collector,
-- its next follow-up and the promise to pay currently standing. Rows appear
-- the first time someone works the customer.
--
-- collection_activities is the account's history. Rows are written once and
-- never edited: a note, call or email logged, a promise to pay with its amount
-- and date, a collector assigned or a follow-up moved.
CREATE TABLE IF NOT EXISTS "collection_accounts" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "customer_id" VARCHAR(100) NOT NULL,
    "collector_id" VARCHAR(100),
    "next_follow_up_on" BIGINT,
    "promise_activity_id" VARCHAR(100),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::BIGINT,
    CONSTRAINT "pk_collection_accounts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_collection_accounts_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_collection_accounts_collector" FOREIGN KEY ("collector_id")
        REFERENCES "users"("id") ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_collection_accounts_customer"
    ON "collection_accounts"("organization_id", "business_unit_id", "customer_id");

CREATE INDEX IF NOT EXISTS "idx_collection_accounts_collector"
    ON "collection_accounts"("organization_id", "business_unit_id", "collector_id", "next_follow_up_on");

CREATE TABLE IF NOT EXISTS "collection_activities" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "customer_id" VARCHAR(100) NOT NULL,
    "kind" VARCHAR(30) NOT NULL,
    "summary" TEXT,
    "contact_name" VARCHAR(200),
    "invoice_id" VARCHAR(100),
    "promised_amount_minor" BIGINT,
    "promised_on" BIGINT,
    "currency_code" VARCHAR(3) NOT NULL DEFAULT 'USD',
    "follow_up_on" BIGINT,
    "collector_id" VARCHAR(100),
    "created_by_id" VARCHAR(100) NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::BIGINT,
    CONSTRAINT "pk_collection_activities" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_collection_activities_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_collection_activities_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id")
        REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE SET NULL ("invoice_id"),
    CONSTRAINT "fk_collection_activities_collector" FOREIGN KEY ("collector_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_collection_activities_created_by" FOREIGN KEY ("created_by_id")
        REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "ck_collection_activities_kind" CHECK ("kind" IN (
        'Note', 'Call', 'Email', 'PromiseToPay', 'CollectorAssigned', 'FollowUpScheduled'
    )),
    CONSTRAINT "ck_collection_activities_promise" CHECK (
        "kind" <> 'PromiseToPay'
        OR ("promised_amount_minor" > 0 AND "promised_on" IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS "idx_collection_activities_customer"
    ON "collection_activities"("organization_id", "business_unit_id", "customer_id", "created_at" DESC);

SELECT trenova_rls.reconcile();
