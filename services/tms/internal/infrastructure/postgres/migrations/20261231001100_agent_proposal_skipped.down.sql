-- PostgreSQL cannot remove a value from an enum. Rows that reached Skipped
-- are read as Rejected by the rollback so nothing points at a value the
-- code no longer knows.
UPDATE "agent_proposals" SET "status" = 'Rejected' WHERE "status" = 'Skipped';
