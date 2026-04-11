CREATE TABLE IF NOT EXISTS "journal_batches"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "batch_number" varchar(50) NOT NULL,
    "batch_type" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL,
    "description" text NOT NULL,
    "accounting_date" bigint NOT NULL,
    "fiscal_year_id" varchar(100) NOT NULL,
    "fiscal_period_id" varchar(100) NOT NULL,
    "entry_count" integer NOT NULL DEFAULT 0,
    "posted_at" bigint,
    "posted_by_id" varchar(100),
    "created_by_id" varchar(100) NOT NULL,
    "updated_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_journal_batches" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_journal_batches_batch_number" UNIQUE ("organization_id", "business_unit_id", "batch_number"),
    CONSTRAINT "fk_journal_batches_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_fiscal_period" FOREIGN KEY ("fiscal_period_id", "organization_id", "business_unit_id") REFERENCES "fiscal_periods"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_posted_by" FOREIGN KEY ("posted_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_batches_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split
ALTER TABLE "journal_entries"
    ADD COLUMN "batch_id" varchar(100);

--bun:split
ALTER TABLE "journal_entries"
    ADD CONSTRAINT "fk_journal_entries_batch"
    FOREIGN KEY ("batch_id", "organization_id", "business_unit_id")
    REFERENCES "journal_batches"("id", "organization_id", "business_unit_id")
    ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_journal_entries_batch_id ON "journal_entries"("batch_id") WHERE "batch_id" IS NOT NULL;

--bun:split
ALTER TABLE "manual_journal_requests"
    ADD CONSTRAINT "fk_manual_journal_requests_posted_batch"
    FOREIGN KEY ("posted_batch_id", "organization_id", "business_unit_id")
    REFERENCES "journal_batches"("id", "organization_id", "business_unit_id")
    ON DELETE SET NULL;
