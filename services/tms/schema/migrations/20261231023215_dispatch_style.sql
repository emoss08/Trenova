-- How much each organization's dispatchers weigh each soft cost, learned from the
-- assignments they make against the standing plan: the planned driver taken, an
-- alternative the plan priced taken instead, or a driver the plan did not price.
-- One row per organization; the weights are multipliers on the organization's own
-- costs, bounded to 0.25-4x, each with the number of decisions that moved it. The
-- observed-through cursor advances with the weights in one write, so a decision is
-- learned once however often the learning runs, and a reset keeps the cursor.
CREATE TABLE IF NOT EXISTS "dispatch_styles"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "weights" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "decisions" bigint NOT NULL DEFAULT 0,
    "approvals" bigint NOT NULL DEFAULT 0,
    "overrides" bigint NOT NULL DEFAULT 0,
    "declines" bigint NOT NULL DEFAULT 0,
    "learning_since" bigint NOT NULL DEFAULT 0,
    "observed_through_at" bigint NOT NULL DEFAULT 0,
    "observed_through_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_dispatch_styles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dispatch_styles_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dispatch_styles_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_dispatch_styles_counts" CHECK (
        "approvals" >= 0 AND "overrides" >= 0 AND "declines" >= 0
        AND "decisions" = "approvals" + "overrides" + "declines"
    ),
    CONSTRAINT "ck_dispatch_styles_weights" CHECK (jsonb_typeof("weights") = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dispatch_styles_tenant"
    ON "dispatch_styles"("organization_id", "business_unit_id");

COMMENT ON TABLE "dispatch_styles" IS 'How much an organization''s dispatchers weigh each soft cost, learned from their approvals, overrides and declines of the standing plan';

-- The planned pairing's cost lines; each alternative carries its own cost and lines
-- in the alternatives document. Learning reads both when a dispatcher assigns.
ALTER TABLE "dispatch_plan_prices"
    ADD COLUMN IF NOT EXISTS "cost_lines" jsonb NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN "dispatch_plan_prices"."cost_lines" IS 'The planned pairing''s cost by component, in cents';
