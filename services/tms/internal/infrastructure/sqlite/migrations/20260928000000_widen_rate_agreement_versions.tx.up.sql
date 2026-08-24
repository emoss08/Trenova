-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260928000000_widen_rate_agreement_versions.tx.up.sql

ALTER TABLE "rate_agreement_versions" ADD COLUMN "party_type" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "customer_id" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "carrier_id" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "code" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "document_id" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "priority" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "agreement_effective_from" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "agreement_effective_to" INTEGER;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "auto_renew" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "renewal_notice_days" INTEGER NOT NULL DEFAULT 30;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "bill_to_customer_id" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "accessorial_terms" TEXT;

--bun:split

ALTER TABLE "rate_agreement_versions" ADD COLUMN "fuel_terms" TEXT;
