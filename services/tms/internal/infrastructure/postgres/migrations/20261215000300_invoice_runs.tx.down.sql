--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP INDEX IF EXISTS "idx_invoices_run";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_invoice_run";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "invoice_run_id";

--bun:split
DROP TABLE IF EXISTS "invoice_run_group_items";

--bun:split
DROP TABLE IF EXISTS "invoice_run_groups";

--bun:split
DROP TABLE IF EXISTS "invoice_runs";

--bun:split
DROP TYPE IF EXISTS "invoice_run_group_status_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_run_source_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_run_status_enum";
