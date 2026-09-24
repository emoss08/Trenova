-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006495_agent_data_access.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "data_access_ceiling" TEXT NOT NULL DEFAULT 'Internal';

--bun:split

UPDATE "agent_definitions"
SET "data_access_ceiling" = 'Restricted'
WHERE "trigger_mode" = 'Chat' OR "template" = 'CashApplication';
