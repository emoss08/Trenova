-- A decided proposal now becomes an active evaluation case on its own, so an
-- organization has a golden set without anyone curating one. The cases the
-- system keeps are marked, held to a number per agent and retired as they age,
-- because a replay reads the records as they are now.
ALTER TABLE "agent_eval_cases"
    ADD COLUMN IF NOT EXISTS "automatic" boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN "agent_eval_cases"."automatic" IS 'Whether the system keeps the case: activated on capture, retired when its agent holds more than the organization keeps or when it ages. Cleared once a person keeps, edits or moves the case, which is then theirs';

COMMENT ON COLUMN "agent_eval_cases"."status" IS 'Candidate until it is activated, by an administrator or on capture when the organization keeps its golden set automatically; Active cases run in every suite; Quarantined cases are kept but not run; Retired cases are kept for their history';

ALTER TABLE "agent_quality_controls"
    ADD COLUMN IF NOT EXISTS "auto_activate" boolean NOT NULL DEFAULT true;

ALTER TABLE "agent_quality_controls"
    ADD COLUMN IF NOT EXISTS "auto_cases_per_agent" integer NOT NULL DEFAULT 100;

ALTER TABLE "agent_quality_controls"
    DROP CONSTRAINT IF EXISTS "ck_agent_quality_controls_auto_cases";

ALTER TABLE "agent_quality_controls"
    ADD CONSTRAINT "ck_agent_quality_controls_auto_cases" CHECK ("auto_cases_per_agent" BETWEEN 1 AND 500);

COMMENT ON COLUMN "agent_quality_controls"."auto_activate" IS 'Whether a proposal a person approved or rejected becomes an active evaluation case on capture, with no one activating it';

COMMENT ON COLUMN "agent_quality_controls"."auto_cases_per_agent" IS 'The most automatic cases one agent keeps active; past it the oldest are retired, a tool at a time so no one tool crowds out the rest';
