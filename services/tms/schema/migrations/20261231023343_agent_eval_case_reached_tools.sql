-- On the Desk an agent can find and call a tool another of the person's agents
-- holds. A replay runs unattended and cannot, so a case whose original did
-- that can never pass. The case keeps which tools those were, and is not
-- activated on its own while it names any.
ALTER TABLE "agent_eval_cases"
    ADD COLUMN IF NOT EXISTS "reached_tools" text[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN "agent_eval_cases"."reached_tools" IS 'The tools the original used that its agent did not hold itself, found through another agent on the Desk. A replay cannot reach them, so a case that names any is left a candidate for a person to judge';
