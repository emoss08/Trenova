CREATE INDEX IF NOT EXISTS "idx_journal_entries_ledger_order"
    ON "journal_entries" ("organization_id", "business_unit_id", "accounting_date" DESC, "id" DESC);
