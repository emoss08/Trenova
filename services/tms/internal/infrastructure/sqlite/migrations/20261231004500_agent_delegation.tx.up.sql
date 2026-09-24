-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231004500_agent_delegation.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "delegate_ids" TEXT;

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "agent_definition_id" TEXT;

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "delegate_call_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_messages_delegate_call"
    ON "assistant_messages" ("organization_id", "business_unit_id", "thread_id", "delegate_call_id")WHERE "delegate_call_id" IS NOT NULL;
