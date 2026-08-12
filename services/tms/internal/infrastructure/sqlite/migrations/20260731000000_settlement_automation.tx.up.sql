-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260731000000_settlement_automation.tx.up.sql

ALTER TABLE "driver_pay_events" ADD COLUMN "on_hold" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "driver_pay_events" ADD COLUMN "hold_reason" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_events_on_hold
    ON driver_pay_events (organization_id, business_unit_id, worker_id)WHERE on_hold = true;

--bun:split

ALTER TABLE "settlement_controls" ADD COLUMN "auto_attach_accruals" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "settlement_controls" ADD COLUMN "auto_post_on_approve" INTEGER NOT NULL DEFAULT 0;
