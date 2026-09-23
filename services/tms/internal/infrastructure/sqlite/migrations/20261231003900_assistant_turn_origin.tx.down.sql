-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003900_assistant_turn_origin.tx.down.sql

ALTER TABLE "assistant_turns" DROP COLUMN "input";

--bun:split

ALTER TABLE "assistant_turns" DROP COLUMN "origin";
