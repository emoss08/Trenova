CREATE TYPE edi_mapping_entity_type_enum AS ENUM(
    'Customer',
    'ServiceType',
    'ShipmentType',
    'FormulaTemplate',
    'Location',
    'Commodity',
    'AccessorialCharge'
);

CREATE TYPE edi_load_tender_transfer_status_enum AS ENUM(
    'Submitted',
    'MappingRequired',
    'PendingApproval',
    'Approved',
    'Rejected',
    'Canceled',
    'Failed'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_mapping_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_mapping_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_mapping_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_mapping_profiles_partner"
    ON "edi_mapping_profiles"("edi_partner_id", "business_unit_id", "organization_id");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_mapping_profile_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "mapping_profile_id" varchar(100) NOT NULL,
    "entity_type" edi_mapping_entity_type_enum NOT NULL,
    "source_id" varchar(100) NOT NULL,
    "source_label" varchar(255),
    "target_id" varchar(100) NOT NULL,
    "target_label" varchar(255),
    "created_by_id" varchar(100),
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_mapping_profile_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_mapping_profile_items_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_mapping_profile_items_profile" FOREIGN KEY ("mapping_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_mapping_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_mapping_profile_items_unique"
    ON "edi_mapping_profile_items"("edi_partner_id", "business_unit_id", "organization_id", "entity_type", "source_id");

CREATE INDEX IF NOT EXISTS "idx_edi_mapping_profile_items_target"
    ON "edi_mapping_profile_items"("target_id", "entity_type", "organization_id");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_load_tender_transfers"(
    "id" varchar(100) NOT NULL,
    "source_organization_id" varchar(100) NOT NULL,
    "source_business_unit_id" varchar(100) NOT NULL,
    "target_organization_id" varchar(100) NOT NULL,
    "target_business_unit_id" varchar(100) NOT NULL,
    "source_partner_id" varchar(100) NOT NULL,
    "target_partner_id" varchar(100) NOT NULL,
    "source_shipment_id" varchar(100) NOT NULL,
    "target_shipment_id" varchar(100),
    "status" edi_load_tender_transfer_status_enum NOT NULL,
    "tender_payload" jsonb NOT NULL,
    "mapping_snapshot" jsonb NOT NULL DEFAULT '[]',
    "rejection_reason" text,
    "failure_reason" text,
    "submitted_by_id" varchar(100),
    "submitted_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    "rejected_by_id" varchar(100),
    "rejected_at" bigint,
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_load_tender_transfers" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_load_tender_transfers_source_partner" FOREIGN KEY ("source_partner_id", "source_business_unit_id", "source_organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_target_partner" FOREIGN KEY ("target_partner_id", "target_business_unit_id", "target_organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_source_shipment" FOREIGN KEY ("source_shipment_id", "source_business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_load_tender_transfers_target_shipment" FOREIGN KEY ("target_shipment_id", "target_business_unit_id", "target_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_open_unique"
    ON "edi_load_tender_transfers"("source_shipment_id", "source_partner_id")
    WHERE "status" NOT IN ('Approved', 'Rejected', 'Canceled', 'Failed');

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_inbound"
    ON "edi_load_tender_transfers"("target_organization_id", "target_business_unit_id", "status", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_outbound"
    ON "edi_load_tender_transfers"("source_organization_id", "source_business_unit_id", "status", "created_at" DESC);
