-- A step after a failed one in a plan never runs. It was neither decided
-- against nor left to expire, and calling it either would say something
-- untrue, so it gets a status of its own. Added outside a transaction
-- because PostgreSQL refuses to use an enum value in the transaction that
-- added it.
ALTER TYPE "agent_proposal_status_enum" ADD VALUE IF NOT EXISTS 'Skipped';
