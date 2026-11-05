-- A palette question is a run owner like the others, and the domain accepts it as a parent, so the
-- check has to as well.
ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_kind";

ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_parent_owner_kind" CHECK (
        "parent_owner_kind" IS NULL OR "parent_owner_kind" IN ('AgentRun', 'AssistantTurn', 'PaletteQuestion')
    ) NOT VALID;
