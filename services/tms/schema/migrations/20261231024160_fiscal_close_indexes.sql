--keel:notx
-- A period's close checks count, for one fiscal period, the journal batches still unposted, the
-- journal reversals still open and the journal entries waiting to be approved or posted. Each
-- partial index holds only the rows still waiting, keyed by period, so a count reads that
-- period's waiting rows and never the tenant's posted history or another period's backlog.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_batches_unposted_period" ON "journal_batches"("organization_id", "business_unit_id", "fiscal_period_id")
WHERE
    "status" <> 'Posted';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_reversals_open_period" ON "journal_reversals"("organization_id", "business_unit_id", "resolved_fiscal_period_id")
WHERE
    "status" IN ('Requested', 'PendingApproval', 'Approved');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_waiting_period" ON "journal_entries"("organization_id", "business_unit_id", "fiscal_period_id")
WHERE
    "status" IN ('Pending', 'Approved')
    AND "is_posted" IS FALSE;
