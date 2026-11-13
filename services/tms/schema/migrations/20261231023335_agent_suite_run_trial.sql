-- A trial run scores an agent's golden set on another provider's model without
-- changing the agent, so a cheaper or newer model can be measured before the
-- agent is moved to it. A trial is never the baseline, the agent's score or a
-- regression: the runs that score the agent are the ones without a trial provider.
ALTER TABLE "agent_suite_runs"
    ADD COLUMN IF NOT EXISTS "trial_provider_id" varchar(100);

ALTER TABLE "agent_suite_runs"
    ADD COLUMN IF NOT EXISTS "trial_label" varchar(300);

ALTER TABLE "agent_suite_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_suite_runs_trial";

ALTER TABLE "agent_suite_runs"
    ADD CONSTRAINT "ck_agent_suite_runs_trial" CHECK (("trial_provider_id" IS NULL) = ("trial_label" IS NULL));

COMMENT ON COLUMN "agent_suite_runs"."trial_provider_id" IS 'The provider a trial run was asked on in place of the agent''s own; empty for a run that scores the agent as it is. Not a foreign key: a trial stays a trial after its provider is deleted';

COMMENT ON COLUMN "agent_suite_runs"."trial_label" IS 'The trial provider''s name and model as they were when the run started, in words';

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "trial_provider_id" varchar(100);

COMMENT ON COLUMN "agent_evaluations"."trial_provider_id" IS 'The provider a trial suite run pins this replay to; empty when the replay runs on the agent''s own provider. A trial replay never counts as the case''s last failure';
