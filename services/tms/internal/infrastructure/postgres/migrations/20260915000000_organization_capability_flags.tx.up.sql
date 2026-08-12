ALTER TABLE "organizations"
    ADD COLUMN IF NOT EXISTS "brokerage_enabled" BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "asset_operations_enabled" BOOLEAN NOT NULL DEFAULT TRUE;

--bun:split
COMMENT ON COLUMN "organizations"."brokerage_enabled" IS 'Whether brokerage surfaces (carriers, routing guides, tendering, rate confirmations, carrier settlements) are presented for this organization; presentation only, not access control';

--bun:split
COMMENT ON COLUMN "organizations"."asset_operations_enabled" IS 'Whether asset operations surfaces (workers, equipment, fleet, driver settlements) are presented for this organization; presentation only, not access control';
