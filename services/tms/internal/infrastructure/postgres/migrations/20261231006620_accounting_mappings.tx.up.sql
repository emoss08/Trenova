-- Where the connection's setup wizard is, and when its reference data was last
-- pulled. Connections made before mapping existed have mapped nothing yet, so
-- they start at the mapping step.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "setup_step" varchar(20) NOT NULL DEFAULT 'Mappings',
    ADD COLUMN IF NOT EXISTS "reference_refresh_started_at" bigint,
    ADD COLUMN IF NOT EXISTS "reference_refreshed_at" bigint,
    ADD COLUMN IF NOT EXISTS "reference_refresh_error" text;

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_setup_step" CHECK ("setup_step" IN ('Mappings', 'Complete'));

--bun:split
-- A cache of the accounting system's own records (accounts, items, customers,
-- vendors, terms, payment methods), used to search, propose and validate
-- mappings without calling the provider on every keystroke.
CREATE TABLE IF NOT EXISTS "accounting_reference_objects"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "external_id" varchar(100) NOT NULL,
    "name" varchar(500) NOT NULL,
    "search_name" varchar(500) NOT NULL,
    "fully_qualified_name" varchar(1000),
    "number" varchar(100),
    "description" text,
    "classification" varchar(50),
    "account_type" varchar(100),
    "account_sub_type" varchar(100),
    "item_type" varchar(50),
    "sub_type" varchar(50),
    "parent_external_id" varchar(100),
    "active" boolean NOT NULL DEFAULT TRUE,
    "currency_code" varchar(3),
    "sync_token" varchar(50),
    "company_name" varchar(500),
    "email" varchar(320),
    "address_line1" varchar(500),
    "city" varchar(255),
    "state" varchar(100),
    "postal_code" varchar(30),
    "income_account_external_id" varchar(100),
    "due_days" integer,
    "is_1099" boolean NOT NULL DEFAULT FALSE,
    "provider_updated_at" bigint,
    "last_seen_at" bigint NOT NULL,
    "removed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_reference_objects" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_reference_objects_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_reference_objects_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_reference_objects_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_reference_objects_kind" CHECK ("kind" IN ('Account', 'Item', 'Customer', 'Vendor', 'Term', 'PaymentMethod'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_reference_objects_external" ON "accounting_reference_objects"("organization_id", "business_unit_id", "connection_id", "kind", "external_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_reference_objects_search" ON "accounting_reference_objects"("organization_id", "business_unit_id", "connection_id", "kind", "search_name");

--bun:split
-- How each Trenova role, line type, charge, customer, carrier, term and payment
-- method maps to a record in the accounting system. Only confirmed rows are
-- used when documents are sent.
CREATE TABLE IF NOT EXISTS "accounting_mappings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "target_type" varchar(30) NOT NULL,
    "trenova_object_id" varchar(100),
    "trenova_key" varchar(50),
    "target_label" varchar(500) NOT NULL,
    "search_label" varchar(500) NOT NULL,
    "provider_kind" varchar(30) NOT NULL,
    "external_id" varchar(100),
    "external_name" varchar(1000),
    "state" varchar(20) NOT NULL DEFAULT 'Unmatched',
    "source" varchar(30),
    "confidence" numeric(5, 4),
    "reason" text,
    "signals" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "confirmed_by_id" varchar(100),
    "confirmed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_mappings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_mappings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_confirmed_by" FOREIGN KEY ("confirmed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_mappings_target_type" CHECK ("target_type" IN ('AccountRole', 'LineType', 'AccessorialCharge', 'ItemRole', 'Customer', 'Carrier', 'PaymentTerm', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_provider_kind" CHECK ("provider_kind" IN ('Account', 'Item', 'Customer', 'Vendor', 'Term', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_state" CHECK ("state" IN ('Unmatched', 'Proposed', 'Confirmed')),
    CONSTRAINT "ck_accounting_mappings_source" CHECK ("source" IS NULL OR "source" IN ('Suggested', 'Model', 'Manual', 'CreatedInProvider', 'Agent')),
    CONSTRAINT "ck_accounting_mappings_target_identity" CHECK (("trenova_object_id" IS NULL) <> ("trenova_key" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_external_when_matched" CHECK (("state" = 'Unmatched') = ("external_id" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_confidence" CHECK ("confidence" IS NULL OR ("confidence" >= 0 AND "confidence" <= 1))
);

--bun:split
-- One row per Trenova target per connection, whether keyed by record or by value.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_mappings_target" ON "accounting_mappings"("organization_id", "business_unit_id", "connection_id", "target_type", COALESCE("trenova_object_id", ''), COALESCE("trenova_key", ''));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_mappings_review" ON "accounting_mappings"("organization_id", "business_unit_id", "connection_id", "target_type", "state");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_mappings_external" ON "accounting_mappings"("organization_id", "business_unit_id", "connection_id", "provider_kind", "external_id") WHERE "external_id" IS NOT NULL;
