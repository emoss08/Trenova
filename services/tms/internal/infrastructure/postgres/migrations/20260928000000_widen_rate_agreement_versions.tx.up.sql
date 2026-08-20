ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "party_type" rate_agreement_party_type_enum;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "customer_id" varchar(100);

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "carrier_id" varchar(100);

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "code" varchar(50) NOT NULL DEFAULT '';

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "document_id" varchar(100);

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "priority" smallint NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "agreement_effective_from" bigint NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "agreement_effective_to" bigint;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "auto_renew" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "renewal_notice_days" smallint NOT NULL DEFAULT 30;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "bill_to_customer_id" varchar(100);

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "accessorial_terms" jsonb;

--bun:split
ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "fuel_terms" jsonb;

--bun:split
-- Backfill copies the agreement's CURRENT header values onto pre-existing
-- version rows: the historical values were never recorded, so this is the only
-- available reading. Rows backfilled here are not audit-grade history for the
-- new columns; versions written after this migration are.
UPDATE
    "rate_agreement_versions" AS v
SET
    "party_type" = a."party_type",
    "customer_id" = a."customer_id",
    "carrier_id" = a."carrier_id",
    "code" = a."code",
    "document_id" = a."document_id",
    "priority" = a."priority",
    "agreement_effective_from" = a."effective_from",
    "agreement_effective_to" = a."effective_to",
    "auto_renew" = a."auto_renew",
    "renewal_notice_days" = a."renewal_notice_days",
    "bill_to_customer_id" = a."bill_to_customer_id"
FROM
    "rate_agreements" AS a
WHERE
    a."id" = v."rate_agreement_id"
    AND a."organization_id" = v."organization_id"
    AND a."business_unit_id" = v."business_unit_id"
    AND v."code" = '';

--bun:split
COMMENT ON COLUMN rate_agreement_versions.agreement_effective_from IS 'The agreement''s own window as it stood; effective_from/effective_to on this table say when this version of the terms governed';

--bun:split
COMMENT ON COLUMN rate_agreement_versions.accessorial_terms IS 'Negotiated accessorial schedule keyed by accessorial charge id; names are resolved at read time, never stored';
