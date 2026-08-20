ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "party_type";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "customer_id";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "carrier_id";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "code";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "document_id";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "priority";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "agreement_effective_from";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "agreement_effective_to";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "auto_renew";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "renewal_notice_days";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "bill_to_customer_id";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "accessorial_terms";

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "fuel_terms";
