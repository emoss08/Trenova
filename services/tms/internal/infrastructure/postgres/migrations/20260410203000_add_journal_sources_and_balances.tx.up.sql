CREATE TABLE IF NOT EXISTS "journal_sources"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "source_object_type" varchar(50) NOT NULL,
    "source_object_id" varchar(100) NOT NULL,
    "source_event_type" varchar(100) NOT NULL,
    "source_document_number" varchar(100),
    "status" varchar(50) NOT NULL,
    "idempotency_key" varchar(200),
    "journal_batch_id" varchar(100) NOT NULL,
    "journal_entry_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_journal_sources" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_journal_sources_idempotency" UNIQUE ("organization_id", "business_unit_id", "idempotency_key"),
    CONSTRAINT "fk_journal_sources_batch" FOREIGN KEY ("journal_batch_id", "organization_id", "business_unit_id") REFERENCES "journal_batches"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_sources_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split
CREATE TABLE IF NOT EXISTS "source_journal_links"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "journal_source_id" varchar(100) NOT NULL,
    "journal_batch_id" varchar(100) NOT NULL,
    "journal_entry_id" varchar(100) NOT NULL,
    "link_role" varchar(50) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_source_journal_links" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_source_journal_links" UNIQUE ("organization_id", "business_unit_id", "journal_source_id", "journal_entry_id", "link_role"),
    CONSTRAINT "fk_source_journal_links_source" FOREIGN KEY ("journal_source_id", "organization_id", "business_unit_id") REFERENCES "journal_sources"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_source_journal_links_batch" FOREIGN KEY ("journal_batch_id", "organization_id", "business_unit_id") REFERENCES "journal_batches"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_source_journal_links_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split
CREATE TABLE IF NOT EXISTS "gl_account_balances_by_period"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "gl_account_id" varchar(100) NOT NULL,
    "fiscal_year_id" varchar(100) NOT NULL,
    "fiscal_period_id" varchar(100) NOT NULL,
    "period_debit_minor" bigint NOT NULL DEFAULT 0,
    "period_credit_minor" bigint NOT NULL DEFAULT 0,
    "net_change_minor" bigint NOT NULL DEFAULT 0,
    "last_journal_entry_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_gl_account_balances_by_period" PRIMARY KEY ("organization_id", "business_unit_id", "gl_account_id", "fiscal_year_id", "fiscal_period_id"),
    CONSTRAINT "fk_gl_balances_account" FOREIGN KEY ("gl_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_period" FOREIGN KEY ("fiscal_period_id", "organization_id", "business_unit_id") REFERENCES "fiscal_periods"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_entry" FOREIGN KEY ("last_journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_journal_sources_object ON "journal_sources"("organization_id", "business_unit_id", "source_object_type", "source_object_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_gl_account_balances_period ON "gl_account_balances_by_period"("organization_id", "business_unit_id", "fiscal_year_id", "fiscal_period_id");
