CREATE TYPE edi_partner_kind_enum AS ENUM(
    'Internal',
    'External'
);

CREATE TYPE edi_partner_role_enum AS ENUM(
    'Customer',
    'Carrier',
    'Broker',
    'Vendor',
    'Shipper',
    'Consignee',
    'BillTo'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_partners"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" edi_partner_kind_enum NOT NULL DEFAULT 'External',
    "status" status_enum NOT NULL DEFAULT 'Active',
    -- Partner lifecycle status. Active partners can be used for EDI communication.
    "code" varchar(100) NOT NULL,
    -- Human-friendly unique partner code within the organization.
    "name" varchar(200) NOT NULL,
    -- Display name for the partner.
    "description" text,
    -- Optional internal description of the partner relationship.
    "internal_organization_id" varchar(100),
    -- For Internal partners, the organization this partner represents.
    "customer_id" varchar(100),
    -- Optional linked Trenova customer record for customer/shipper/bill-to relationships.
    "default_transport_id" varchar(100),
    -- Preferred transport configuration for sending/receiving messages with this partner.
    "default_mapping_profile_id" varchar(100),
    -- Default mapping profile used to translate partner data into Trenova records.
    "default_validation_profile_id" varchar(100),
    -- Default validation profile used to validate messages from/to this partner.
    "timezone" varchar(100),
    -- Partner’s preferred timezone for appointments, cutoffs, and scheduling rules.
    "country" varchar(2) NOT NULL DEFAULT 'US',
    -- ISO-3166 alpha-2 country code for the partner.
    "contact_name" varchar(150),
    -- Primary operational or EDI contact name.
    "contact_email" varchar(255),
    -- Primary operational or EDI contact email.
    "contact_phone" varchar(30),
    -- Primary operational or EDI contact phone number.
    "enabled_for_inbound" boolean NOT NULL DEFAULT TRUE,
    -- Whether this partner can send messages into Trenova.
    "enabled_for_outbound" boolean NOT NULL DEFAULT TRUE,
    -- Whether Trenova can send messages to this partner.
    "settings" jsonb NOT NULL DEFAULT '{}',
    -- Partner-specific configuration that does not deserve a dedicated column yet.
    "version" bigint NOT NULL DEFAULT 0,
    -- Optimistic-locking version for safe concurrent updates.
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    -- Unix timestamp when the partner was created.
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    -- Unix timestamp when the partner was last updated.
    CONSTRAINT "pk_edi_partners" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_partners_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partners_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partners_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_partners_internal_org" FOREIGN KEY ("internal_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX "idx_edi_partners_code_org" ON "edi_partners"(lower("code"), "organization_id");

CREATE UNIQUE INDEX "idx_edi_partners_name_org" ON "edi_partners"(lower("name"), "organization_id");

CREATE UNIQUE INDEX "idx_edi_partners_internal_relationship_org_bu" ON "edi_partners"("organization_id", "business_unit_id", "internal_organization_id")
    WHERE "kind" = 'Internal' AND "internal_organization_id" IS NOT NULL;

CREATE INDEX "idx_edi_partners_created_updated" ON "edi_partners"("created_at", "updated_at");

--bun:split
ALTER TABLE "edi_partners"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'B') || setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("kind"), '')), 'C') || setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'C')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_partners_search" ON "edi_partners" USING GIN("search_vector");

CREATE STATISTICS IF NOT EXISTS "edi_partners_kind_status_org_stats"(dependencies) ON kind, status, organization_id FROM edi_partners;

CREATE STATISTICS IF NOT EXISTS "edi_partners_kind_status_bu_stats"(dependencies) ON kind, status, business_unit_id FROM edi_partners;
