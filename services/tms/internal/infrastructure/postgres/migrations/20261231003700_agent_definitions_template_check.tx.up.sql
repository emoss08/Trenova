-- The template column is an enum of the agentdefinition domain, and
-- docs/databases.md is explicit that enum values are not enforced with a CHECK:
-- the set grows in code, and a constraint written once lags behind it. This one
-- had fallen eight templates behind the sixteen the domain defines, so seeding
-- the intake desk failed and every later template would have failed the same way.
ALTER TABLE "agent_definitions" DROP CONSTRAINT IF EXISTS "ck_agent_definitions_template";
