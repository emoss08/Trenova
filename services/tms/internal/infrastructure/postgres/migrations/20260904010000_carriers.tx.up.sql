CREATE TABLE IF NOT EXISTS "carriers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "state_id" varchar(100),
    "remit_state_id" varchar(100),
    "status" varchar(50) NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "name" varchar(255) NOT NULL,
    "dba_name" varchar(255),
    "carrier_type" varchar(50) NOT NULL DEFAULT 'Common',
    "dot_number" varchar(12),
    "mc_number" varchar(12),
    "scac" varchar(4),
    "compliance_status" varchar(50) NOT NULL DEFAULT 'Pending',
    "safety_rating" varchar(50) NOT NULL DEFAULT 'NotRated',
    "qualified_at" bigint,
    "disqualified_reason" text,
    "tax_id" varchar(20),
    "tax_id_type" varchar(10),
    "w9_on_file" boolean NOT NULL DEFAULT FALSE,
    "is_1099_eligible" boolean NOT NULL DEFAULT FALSE,
    "payment_method" varchar(50) NOT NULL DEFAULT 'Check',
    "payment_term_days" integer NOT NULL DEFAULT 30,
    "remit_to_name" varchar(255),
    "remit_address_line_1" varchar(150),
    "remit_address_line_2" varchar(150),
    "remit_city" varchar(100),
    "remit_postal_code" varchar(10),
    "address_line_1" varchar(150),
    "address_line_2" varchar(150),
    "city" varchar(100),
    "postal_code" varchar(10),
    "phone" varchar(20),
    "email" varchar(255),
    "external_id" text,
    "notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carriers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carriers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carriers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carriers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carriers_remit_state" FOREIGN KEY ("remit_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carriers_payment_term_days" CHECK ("payment_term_days" >= 0 AND "payment_term_days" <= 365)
);

--bun:split
CREATE UNIQUE INDEX "idx_carriers_code" ON "carriers"(lower("code"), "organization_id");

--bun:split
CREATE UNIQUE INDEX "idx_carriers_dot_number" ON "carriers"("dot_number", "organization_id")
WHERE
    "dot_number" IS NOT NULL;

--bun:split
CREATE UNIQUE INDEX "idx_carriers_scac" ON "carriers"("scac", "organization_id")
WHERE
    "scac" IS NOT NULL;

--bun:split
CREATE INDEX "idx_carriers_name" ON "carriers"("name");

--bun:split
CREATE INDEX "idx_carriers_business_unit_organization" ON "carriers"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX "idx_carriers_status" ON "carriers"("status");

--bun:split
CREATE INDEX "idx_carriers_compliance_status" ON "carriers"("compliance_status", "organization_id");

--bun:split
CREATE INDEX "idx_carriers_created_updated" ON "carriers"("created_at", "updated_at");

--bun:split
COMMENT ON TABLE "carriers" IS 'Stores external carriers used to cover brokered shipment moves';

--bun:split
ALTER TABLE "carriers"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("dba_name", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("dot_number", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("scac", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_carriers_search ON carriers USING GIN(search_vector);

--bun:split
CREATE STATISTICS IF NOT EXISTS carriers_org_bu_stats (dependencies)
    ON organization_id, business_unit_id FROM carriers;

--bun:split
CREATE INDEX IF NOT EXISTS idx_carriers_trgm_code ON carriers USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carriers_trgm_name ON carriers USING gin(name gin_trgm_ops);

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_contacts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "title" varchar(100),
    "email" varchar(255),
    "phone" varchar(20),
    "is_primary" boolean NOT NULL DEFAULT FALSE,
    "receives_rate_confirmations" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_contacts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_contacts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_contacts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_contacts_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX "idx_carrier_contacts_carrier" ON "carrier_contacts"("carrier_id", "organization_id");

--bun:split
COMMENT ON TABLE "carrier_contacts" IS 'Stores contact people for carriers, including rate confirmation recipients';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_insurance_policies"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "policy_type" varchar(50) NOT NULL,
    "policy_number" varchar(100) NOT NULL,
    "provider_name" varchar(255) NOT NULL,
    "coverage_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "effective_date" bigint NOT NULL,
    "expiration_date" bigint NOT NULL,
    "is_verified" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_insurance_policies" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_insurance_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_insurance_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_insurance_policies_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_insurance_policies_dates" CHECK ("expiration_date" > "effective_date"),
    CONSTRAINT "chk_carrier_insurance_policies_coverage" CHECK ("coverage_amount" >= 0)
);

--bun:split
CREATE INDEX "idx_carrier_insurance_policies_carrier" ON "carrier_insurance_policies"("carrier_id", "organization_id");

--bun:split
CREATE INDEX "idx_carrier_insurance_policies_expiration" ON "carrier_insurance_policies"("organization_id", "expiration_date");

--bun:split
COMMENT ON TABLE "carrier_insurance_policies" IS 'Stores carrier insurance policies with coverage and expiration tracking';
