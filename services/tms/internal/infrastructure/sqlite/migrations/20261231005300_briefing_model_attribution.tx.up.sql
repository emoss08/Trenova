-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005300_briefing_model_attribution.tx.up.sql

ALTER TABLE "assistant_briefings" ADD COLUMN "model_identifier" TEXT;

--bun:split

ALTER TABLE "assistant_briefings" ADD COLUMN "provider_id" TEXT;
