-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261229000600_agent_definition_identity.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "icon" TEXT;

--bun:split

ALTER TABLE "agent_definitions" ADD COLUMN "accent" TEXT;
