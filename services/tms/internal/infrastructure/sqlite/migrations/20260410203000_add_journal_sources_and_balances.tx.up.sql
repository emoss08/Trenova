-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260410203000_add_journal_sources_and_balances.tx.up.sql

CREATE TABLE IF NOT EXISTS "journal_sources"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "source_object_type" TEXT NOT NULL,
    "source_object_id" TEXT NOT NULL,
    "source_event_type" TEXT NOT NULL,
    "source_document_number" TEXT,
    "status" TEXT NOT NULL,
    "idempotency_key" TEXT,
    "journal_batch_id" TEXT NOT NULL,
    "journal_entry_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_journal_sources" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_journal_sources_idempotency" UNIQUE ("organization_id", "business_unit_id", "idempotency_key"),
    CONSTRAINT "fk_journal_sources_batch" FOREIGN KEY ("journal_batch_id", "organization_id", "business_unit_id") REFERENCES "journal_batches"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_sources_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS "source_journal_links"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "journal_source_id" TEXT NOT NULL,
    "journal_batch_id" TEXT NOT NULL,
    "journal_entry_id" TEXT NOT NULL,
    "link_role" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_source_journal_links" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_source_journal_links" UNIQUE ("organization_id", "business_unit_id", "journal_source_id", "journal_entry_id", "link_role"),
    CONSTRAINT "fk_source_journal_links_source" FOREIGN KEY ("journal_source_id", "organization_id", "business_unit_id") REFERENCES "journal_sources"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_source_journal_links_batch" FOREIGN KEY ("journal_batch_id", "organization_id", "business_unit_id") REFERENCES "journal_batches"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_source_journal_links_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS "gl_account_balances_by_period"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "gl_account_id" TEXT NOT NULL,
    "fiscal_year_id" TEXT NOT NULL,
    "fiscal_period_id" TEXT NOT NULL,
    "period_debit_minor" INTEGER NOT NULL DEFAULT 0,
    "period_credit_minor" INTEGER NOT NULL DEFAULT 0,
    "net_change_minor" INTEGER NOT NULL DEFAULT 0,
    "last_journal_entry_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_gl_account_balances_by_period" PRIMARY KEY ("organization_id", "business_unit_id", "gl_account_id", "fiscal_year_id", "fiscal_period_id"),
    CONSTRAINT "fk_gl_balances_account" FOREIGN KEY ("gl_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_period" FOREIGN KEY ("fiscal_period_id", "organization_id", "business_unit_id") REFERENCES "fiscal_periods"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_balances_entry" FOREIGN KEY ("last_journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_sources_object ON "journal_sources" ("organization_id", "business_unit_id", "source_object_type", "source_object_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_account_balances_period ON "gl_account_balances_by_period" ("organization_id", "business_unit_id", "fiscal_year_id", "fiscal_period_id");
