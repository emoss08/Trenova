-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260904010000_carriers.tx.up.sql

CREATE TABLE IF NOT EXISTS "carriers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "state_id" TEXT,
    "remit_state_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "dba_name" TEXT,
    "carrier_type" TEXT NOT NULL DEFAULT 'Common',
    "dot_number" TEXT,
    "mc_number" TEXT,
    "scac" TEXT,
    "compliance_status" TEXT NOT NULL DEFAULT 'Pending',
    "safety_rating" TEXT NOT NULL DEFAULT 'NotRated',
    "qualified_at" INTEGER,
    "disqualified_reason" TEXT,
    "tax_id" TEXT,
    "tax_id_type" TEXT,
    "w9_on_file" INTEGER NOT NULL DEFAULT 0,
    "is_1099_eligible" INTEGER NOT NULL DEFAULT 0,
    "payment_method" TEXT NOT NULL DEFAULT 'Check',
    "payment_term_days" INTEGER NOT NULL DEFAULT 30,
    "remit_to_name" TEXT,
    "remit_address_line_1" TEXT,
    "remit_address_line_2" TEXT,
    "remit_city" TEXT,
    "remit_postal_code" TEXT,
    "address_line_1" TEXT,
    "address_line_2" TEXT,
    "city" TEXT,
    "postal_code" TEXT,
    "phone" TEXT,
    "email" TEXT,
    "external_id" TEXT,
    "notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carriers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carriers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carriers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carriers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carriers_remit_state" FOREIGN KEY ("remit_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carriers_payment_term_days" CHECK ("payment_term_days" >= 0 AND "payment_term_days" <= 365)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_carriers_code" ON "carriers" (lower("code"), "organization_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_carriers_dot_number" ON "carriers" ("dot_number", "organization_id")WHERE
    "dot_number" IS NOT NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_carriers_scac" ON "carriers" ("scac", "organization_id")WHERE
    "scac" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_name" ON "carriers" ("name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_business_unit_organization" ON "carriers" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_status" ON "carriers" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_compliance_status" ON "carriers" ("compliance_status", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_created_updated" ON "carriers" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_contacts"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "title" TEXT,
    "email" TEXT,
    "phone" TEXT,
    "is_primary" INTEGER NOT NULL DEFAULT 0,
    "receives_rate_confirmations" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_contacts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_contacts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_contacts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_contacts_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_contacts_carrier" ON "carrier_contacts" ("carrier_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_insurance_policies"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "policy_type" TEXT NOT NULL,
    "policy_number" TEXT NOT NULL,
    "provider_name" TEXT NOT NULL,
    "coverage_amount" REAL NOT NULL DEFAULT 0,
    "effective_date" INTEGER NOT NULL,
    "expiration_date" INTEGER NOT NULL,
    "is_verified" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_insurance_policies" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_insurance_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_insurance_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_insurance_policies_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_insurance_policies_dates" CHECK ("expiration_date" > "effective_date"),
    CONSTRAINT "chk_carrier_insurance_policies_coverage" CHECK ("coverage_amount" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_insurance_policies_carrier" ON "carrier_insurance_policies" ("carrier_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_insurance_policies_expiration" ON "carrier_insurance_policies" ("organization_id", "expiration_date");
