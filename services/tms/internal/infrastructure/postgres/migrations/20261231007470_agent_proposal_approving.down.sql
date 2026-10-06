-- An enum value cannot be dropped; an approval still waiting to commit goes
-- back to waiting on its decider.
UPDATE "agent_proposals" SET "status" = 'Pending' WHERE "status" = 'Approving';
