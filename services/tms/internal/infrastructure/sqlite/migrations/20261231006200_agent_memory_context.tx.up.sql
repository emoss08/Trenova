-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006200_agent_memory_context.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "memory_token_budget" INTEGER;
