-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260226143000_add_worker_pto_chart_index.tx.up.sql

CREATE INDEX IF NOT EXISTS idx_worker_pto_chart_approved
    ON "worker_pto" (
        "organization_id",
        "business_unit_id",
        "start_date",
        "end_date",
        "type",
        "worker_id"
    )WHERE "status" = 'Approved';
