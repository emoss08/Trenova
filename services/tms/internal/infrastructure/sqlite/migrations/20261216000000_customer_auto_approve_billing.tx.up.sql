-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261216000000_customer_auto_approve_billing.tx.up.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "auto_approve" INTEGER NOT NULL DEFAULT 0;
