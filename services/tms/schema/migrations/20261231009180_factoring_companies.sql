-- Factoring: an organization that sells its receivables names the factor
-- (factoring_companies) and the customers whose invoices are assigned to it
-- (customer_billing_profiles.factoring_company_id). An invoice records the
-- factor it was assigned to when it posted, so its remit-to and notice of
-- assignment keep naming that factor after the customer moves.
CREATE TABLE IF NOT EXISTS "factoring_companies" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" VARCHAR(10) NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "address_line_1" VARCHAR(150) NOT NULL,
    "address_line_2" VARCHAR(150),
    "city" VARCHAR(100) NOT NULL,
    "state_id" VARCHAR(100) NOT NULL,
    "postal_code" us_postal_code NOT NULL,
    "email" VARCHAR(255),
    "phone_number" VARCHAR(30),
    "remittance_instructions" TEXT,
    "notice_of_assignment" TEXT,
    "notes" TEXT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_factoring_companies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_companies_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_companies_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_companies_state" FOREIGN KEY ("state_id")
        REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "ck_factoring_companies_remittance_length" CHECK (char_length("remittance_instructions") <= 2000),
    CONSTRAINT "ck_factoring_companies_notice_length" CHECK (char_length("notice_of_assignment") <= 2000)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_companies_code"
    ON "factoring_companies"("organization_id", "business_unit_id", lower("code"));

CREATE INDEX IF NOT EXISTS "idx_factoring_companies_status"
    ON "factoring_companies"("organization_id", "business_unit_id", "status");

ALTER TABLE "factoring_companies"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("city", '')), 'C') ||
        setweight(immutable_to_tsvector('simple', COALESCE("email", '')), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS "idx_factoring_companies_search"
    ON "factoring_companies" USING GIN("search_vector");

ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "factoring_company_id" VARCHAR(100);

ALTER TABLE "customer_billing_profiles"
    DROP CONSTRAINT IF EXISTS "fk_customer_billing_profiles_factoring_company";

ALTER TABLE "customer_billing_profiles"
    ADD CONSTRAINT "fk_customer_billing_profiles_factoring_company"
        FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id")
        REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id")
        ON UPDATE NO ACTION ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_factoring_company"
    ON "customer_billing_profiles"("organization_id", "business_unit_id", "factoring_company_id")
    WHERE "factoring_company_id" IS NOT NULL;

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "factoring_company_id" VARCHAR(100);

ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_factoring_company";

ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_factoring_company"
        FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id")
        REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id")
        ON UPDATE NO ACTION ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS "idx_invoices_factoring_company"
    ON "invoices"("organization_id", "business_unit_id", "factoring_company_id")
    WHERE "factoring_company_id" IS NOT NULL;

SELECT trenova_rls.reconcile();
