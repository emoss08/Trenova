--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP INDEX IF EXISTS idx_invoices_tenant_scope;

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "scope",
    DROP COLUMN IF EXISTS "period_start",
    DROP COLUMN IF EXISTS "period_end",
    DROP COLUMN IF EXISTS "shipment_count";

--bun:split
DROP TYPE IF EXISTS "invoice_scope_enum";
