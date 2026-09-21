-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000600_assistant_reasoning.tx.up.sql

ALTER TABLE "ai_providers" ADD COLUMN "reasoning_effort" TEXT NOT NULL DEFAULT 'Off';

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "reasoning" TEXT;
