-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260406200000_customer_billing_profile_biller.tx.up.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "default_biller_id" TEXT;
