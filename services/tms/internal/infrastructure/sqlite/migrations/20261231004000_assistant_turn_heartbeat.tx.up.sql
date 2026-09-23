-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231004000_assistant_turn_heartbeat.tx.up.sql

ALTER TABLE "assistant_turns" ADD COLUMN "heartbeat_at" INTEGER;

--bun:split

UPDATE "assistant_turns"
SET "heartbeat_at" = COALESCE("started_at", "created_at")
WHERE "heartbeat_at" IS NULL;
