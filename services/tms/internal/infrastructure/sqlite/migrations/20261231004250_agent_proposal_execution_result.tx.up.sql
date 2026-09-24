-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231004250_agent_proposal_execution_result.tx.up.sql

ALTER TABLE "agent_proposals" ADD COLUMN "execution_result" TEXT;
