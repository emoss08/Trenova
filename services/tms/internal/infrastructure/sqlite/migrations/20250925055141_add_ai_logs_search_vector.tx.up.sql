-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250925055141_add_ai_logs_search_vector.tx.up.sql

CREATE INDEX IF NOT EXISTS idx_ai_logs_user_id ON "ai_logs" ("user_id");
