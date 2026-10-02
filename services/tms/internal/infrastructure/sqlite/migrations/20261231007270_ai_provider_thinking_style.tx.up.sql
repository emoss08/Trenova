-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231007270_ai_provider_thinking_style.tx.up.sql

ALTER TABLE "ai_providers" ADD COLUMN "thinking_style" TEXT NOT NULL DEFAULT 'Auto';
