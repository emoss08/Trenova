-- A finished run keeps a bounded sample of its result (the columns, the first
-- rows and the grand totals) so the result can be read where it is shown
-- without opening the stored rows file. It is cleared when the run expires.
ALTER TABLE "report_runs"
    ADD COLUMN IF NOT EXISTS "sample" JSONB;
