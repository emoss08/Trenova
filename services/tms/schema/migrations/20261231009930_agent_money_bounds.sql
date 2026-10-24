-- What an agent may move in money on its own. A money write runs without a
-- person only when its tool proves the call is within these bounds; the
-- defaults are deliberately low, and an organization raises them in AI Control.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "agent_money_call_ceiling_minor" BIGINT NOT NULL DEFAULT 100000,
    ADD COLUMN IF NOT EXISTS "agent_money_run_calls" INTEGER NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS "agent_money_run_amount_minor" BIGINT NOT NULL DEFAULT 500000,
    ADD COLUMN IF NOT EXISTS "agent_money_day_calls" INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS "agent_money_day_amount_minor" BIGINT NOT NULL DEFAULT 2500000;

ALTER TABLE "agent_controls"
    ADD CONSTRAINT "ck_agent_controls_agent_money_bounds" CHECK (
        "agent_money_call_ceiling_minor" >= 0
        AND "agent_money_run_calls" >= 0
        AND "agent_money_run_amount_minor" >= 0
        AND "agent_money_day_calls" >= 0
        AND "agent_money_day_amount_minor" >= 0
    );

COMMENT ON COLUMN "agent_controls"."agent_money_call_ceiling_minor" IS 'Most one automatic money write by an agent may move, in minor units';

COMMENT ON COLUMN "agent_controls"."agent_money_run_calls" IS 'Most automatic money writes one agent run or turn may make';

COMMENT ON COLUMN "agent_controls"."agent_money_run_amount_minor" IS 'Most one agent run or turn may move in automatic money writes, in minor units';

COMMENT ON COLUMN "agent_controls"."agent_money_day_calls" IS 'Most automatic money writes the organization''s agents may make in a day, in its own timezone';

COMMENT ON COLUMN "agent_controls"."agent_money_day_amount_minor" IS 'Most the organization''s agents may move in automatic money writes in a day, in minor units';

-- What an automatic money write moved, and in which currency, counted against
-- those caps. Set only on a write that ran on its own inside its bounds; the
-- caps count only writes in the organization's functional currency.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "money_minor" BIGINT,
    ADD COLUMN IF NOT EXISTS "money_currency" VARCHAR(3);

ALTER TABLE "agent_proposals"
    ADD CONSTRAINT "ck_agent_proposals_money_currency" CHECK (
        ("money_minor" IS NULL) = ("money_currency" IS NULL)
        AND ("money_currency" IS NULL OR "money_currency" = upper("money_currency"))
    );

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_automatic_money"
    ON "agent_proposals" ("organization_id", "business_unit_id", "executed_at")
    WHERE "money_minor" IS NOT NULL;

-- Whether an agent added a settlement adjustment line, so an agent may take
-- back only what an agent put there.
ALTER TABLE "driver_settlement_lines"
    ADD COLUMN IF NOT EXISTS "added_by_agent" BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE "carrier_settlement_lines"
    ADD COLUMN IF NOT EXISTS "added_by_agent" BOOLEAN NOT NULL DEFAULT FALSE;
