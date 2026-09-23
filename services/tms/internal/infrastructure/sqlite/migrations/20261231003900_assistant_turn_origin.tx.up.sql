-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite has no
-- ALTER TABLE ADD CONSTRAINT, so the origin check is enforced by the
-- application here, as the domain's enum already does.
-- Source: 20261231003900_assistant_turn_origin.tx.up.sql

ALTER TABLE "assistant_turns" ADD COLUMN "origin" TEXT NOT NULL DEFAULT 'Person';

--bun:split

ALTER TABLE "assistant_turns" ADD COLUMN "input" TEXT;
