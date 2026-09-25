-- What a write would do, as it was proposed and as it was approved. The
-- baseline is taken when the proposal is filed, in the snapshot that pins its
-- target's version, and is kept apart from the proposal so the workflow never
-- carries it. It is written before the proposal row exists, so it names the
-- proposal without a foreign key, and a baseline whose proposal was never
-- filed is purged by the expiry sweep. A decision records the preview its
-- decider was shown and whether they approved that exact preview.
CREATE TABLE IF NOT EXISTS "agent_proposal_baselines"(
    "proposal_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "tool_name" VARCHAR(100) NOT NULL,
    "preview" JSONB NOT NULL,
    "target_version" BIGINT,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_proposal_baselines" PRIMARY KEY ("proposal_id", "organization_id", "business_unit_id"),
    CONSTRAINT "ck_agent_proposal_baselines_target_version" CHECK ("target_version" IS NULL OR "target_version" >= 0),
    CONSTRAINT "fk_agent_proposal_baselines_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposal_baselines_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "agent_proposal_baselines" IS 'What a proposed write would have done when it was proposed, unfiltered but for Confidential values, read only to say which values moved since';

COMMENT ON COLUMN "agent_proposal_baselines"."proposal_id" IS 'The proposal the baseline was taken for; the proposal row may not exist yet, or ever, when its filing was retried';

COMMENT ON COLUMN "agent_proposal_baselines"."preview" IS 'The tool''s preview at filing time, bounded to 32 KiB';

COMMENT ON COLUMN "agent_proposal_baselines"."target_version" IS 'The target record''s version in the same snapshot, when the tool names one';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposal_baselines_created_at" ON "agent_proposal_baselines"("created_at");

--bun:split

ALTER TABLE "agent_decisions"
    ADD COLUMN IF NOT EXISTS "preview" jsonb,
    ADD COLUMN IF NOT EXISTS "preview_digest" varchar(64),
    ADD COLUMN IF NOT EXISTS "preview_reviewed" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "preview_target_version" bigint;

COMMENT ON COLUMN "agent_decisions"."preview" IS 'What the decider was shown of the write, filtered for them, when it was approved';

COMMENT ON COLUMN "agent_decisions"."preview_digest" IS 'SHA-256 of that preview''s canonical form, as 64 lowercase hex characters';

COMMENT ON COLUMN "agent_decisions"."preview_reviewed" IS 'The decision named the digest of the preview it recorded, so the decider approved what they were shown';

COMMENT ON COLUMN "agent_decisions"."preview_target_version" IS 'The target record''s version the recorded preview was read at';

--bun:split

ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_preview_digest";

--bun:split

ALTER TABLE "agent_decisions"
    ADD CONSTRAINT "ck_agent_decisions_preview_digest" CHECK (
        "preview_digest" IS NULL OR "preview_digest" ~ '^[0-9a-f]{64}$'
    ) NOT VALID;

--bun:split

ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_preview_reviewed";

--bun:split

ALTER TABLE "agent_decisions"
    ADD CONSTRAINT "ck_agent_decisions_preview_reviewed" CHECK (
        NOT "preview_reviewed" OR "preview_digest" IS NOT NULL
    ) NOT VALID;
