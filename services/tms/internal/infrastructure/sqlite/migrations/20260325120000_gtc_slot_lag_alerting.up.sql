-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260325120000_gtc_slot_lag_alerting.up.sql

CREATE TABLE IF NOT EXISTS gtc_slot_alerts (
    "id" INTEGER PRIMARY KEY,
    "slot_name" TEXT NOT NULL,
    "lag_bytes" INTEGER NOT NULL,
    "checked_at" TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
