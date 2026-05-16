CREATE TYPE edi_connection_method_enum AS ENUM(
    'Internal',
    'AS2',
    'SFTP',
    'VAN'
);

CREATE TYPE edi_connection_status_enum AS ENUM(
    'PendingAcceptance',
    'Active',
    'Suspended',
    'Rejected',
    'Revoked'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_connections"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "source_organization_id" varchar(100) NOT NULL,
    "target_organization_id" varchar(100) NOT NULL,
    "source_partner_id" varchar(100),
    "target_partner_id" varchar(100),
    "method" edi_connection_method_enum NOT NULL,
    "status" edi_connection_status_enum NOT NULL DEFAULT 'PendingAcceptance',
    "capabilities" jsonb NOT NULL DEFAULT '{}',
    "source_partner_config" jsonb NOT NULL DEFAULT '{}',
    "target_partner_config" jsonb NOT NULL DEFAULT '{}',
    "requested_by_id" varchar(100),
    "requested_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "accepted_by_id" varchar(100),
    "accepted_at" bigint,
    "rejected_by_id" varchar(100),
    "rejected_at" bigint,
    "rejection_reason" text,
    "suspended_by_id" varchar(100),
    "suspended_at" bigint,
    "revoked_by_id" varchar(100),
    "revoked_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_connections" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_connections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_connections_source_org" FOREIGN KEY ("source_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_connections_target_org" FOREIGN KEY ("target_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_connections_distinct_orgs" CHECK ("source_organization_id" <> "target_organization_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_connections_internal_open"
    ON "edi_connections"("business_unit_id", LEAST("source_organization_id", "target_organization_id"), GREATEST("source_organization_id", "target_organization_id"), "method")
    WHERE "method" = 'Internal' AND "status" IN ('PendingAcceptance', 'Active', 'Suspended');

CREATE INDEX IF NOT EXISTS "idx_edi_connections_source"
    ON "edi_connections"("source_organization_id", "business_unit_id", "status", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_edi_connections_target"
    ON "edi_connections"("target_organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_communication_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_connection_id" varchar(100),
    "edi_partner_id" varchar(100),
    "method" edi_connection_method_enum NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(200) NOT NULL,
    "description" text,
    "config" jsonb NOT NULL DEFAULT '{}',
    "encrypted_secrets" jsonb NOT NULL DEFAULT '{}',
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("method"), '')), 'C') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'C')
    ) STORED,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_communication_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_communication_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_communication_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_communication_profiles_connection" FOREIGN KEY ("edi_connection_id") REFERENCES "edi_connections"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_communication_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("edi_partner_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_communication_profiles_name_org"
    ON "edi_communication_profiles"(lower("name"), "business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_edi_communication_profiles_partner"
    ON "edi_communication_profiles"("edi_partner_id", "business_unit_id", "organization_id", "status");

CREATE INDEX IF NOT EXISTS "idx_edi_communication_profiles_search"
    ON "edi_communication_profiles" USING GIN("search_vector");

--bun:split
ALTER TABLE "edi_partners"
    ADD COLUMN "edi_connection_id" varchar(100);

ALTER TABLE "edi_partners"
    ADD CONSTRAINT "fk_edi_partners_connection" FOREIGN KEY ("edi_connection_id") REFERENCES "edi_connections"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

ALTER TABLE "edi_partners"
    ADD CONSTRAINT "fk_edi_partners_default_transport" FOREIGN KEY ("default_transport_id", "business_unit_id", "organization_id") REFERENCES "edi_communication_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("default_transport_id");

--bun:split
CREATE TYPE shipment_tender_status_enum AS ENUM(
    'Tendered',
    'Accepted',
    'Rejected',
    'Expired',
    'Canceled'
);

CREATE TYPE shipment_entry_method_enum AS ENUM(
    'Manual',
    'EDI'
);

CREATE TYPE edi_shipment_sync_policy_enum AS ENUM(
    'ManualReview',
    'AutoOperational',
    'AutoAllSafe',
    'ReadOnly'
);

CREATE TYPE edi_shipment_link_status_enum AS ENUM(
    'Active',
    'Suspended',
    'Closed'
);

CREATE TYPE edi_transfer_change_direction_enum AS ENUM(
    'SourceToTarget',
    'TargetToSource'
);

CREATE TYPE edi_transfer_change_status_enum AS ENUM(
    'PendingReview',
    'Applied',
    'Rejected',
    'Failed',
    'Ignored'
);

CREATE TYPE edi_transfer_change_conflict_status_enum AS ENUM(
    'None',
    'Conflict',
    'Resolved'
);

--bun:split
ALTER TYPE edi_load_tender_transfer_status_enum ADD VALUE IF NOT EXISTS 'Expired';

ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "tender_status" shipment_tender_status_enum,
    ADD COLUMN IF NOT EXISTS "entry_method" shipment_entry_method_enum NOT NULL DEFAULT 'Manual';

CREATE INDEX IF NOT EXISTS "idx_shipments_tender_status"
    ON "shipments"("business_unit_id", "organization_id", "tender_status")
    WHERE "tender_status" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_shipments_entry_method"
    ON "shipments"("business_unit_id", "organization_id", "entry_method");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_shipment_links"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "source_organization_id" varchar(100) NOT NULL,
    "target_organization_id" varchar(100) NOT NULL,
    "source_shipment_id" varchar(100) NOT NULL,
    "target_shipment_id" varchar(100) NOT NULL,
    "tender_transfer_id" varchar(100) NOT NULL,
    "originating_message_id" varchar(100),
    "sync_policy" edi_shipment_sync_policy_enum NOT NULL DEFAULT 'AutoOperational',
    "field_ownership" jsonb NOT NULL DEFAULT '{}',
    "status" edi_shipment_link_status_enum NOT NULL DEFAULT 'Active',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_shipment_links" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_shipment_links_source_org" FOREIGN KEY ("source_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_target_org" FOREIGN KEY ("target_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_source_shipment" FOREIGN KEY ("source_shipment_id", "business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_target_shipment" FOREIGN KEY ("target_shipment_id", "business_unit_id", "target_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_shipment_links_transfer" FOREIGN KEY ("tender_transfer_id") REFERENCES "edi_load_tender_transfers"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_shipment_links_transfer"
    ON "edi_shipment_links"("tender_transfer_id");

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_shipment_links_source_target"
    ON "edi_shipment_links"("business_unit_id", "source_organization_id", "source_shipment_id", "target_organization_id", "target_shipment_id");

CREATE INDEX IF NOT EXISTS "idx_edi_shipment_links_source_lookup"
    ON "edi_shipment_links"("business_unit_id", "source_organization_id", "source_shipment_id", "status");

CREATE INDEX IF NOT EXISTS "idx_edi_shipment_links_target_lookup"
    ON "edi_shipment_links"("business_unit_id", "target_organization_id", "target_shipment_id", "status");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_transfer_changes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_link_id" varchar(100) NOT NULL,
    "direction" edi_transfer_change_direction_enum NOT NULL,
    "change_type" varchar(100) NOT NULL,
    "status" edi_transfer_change_status_enum NOT NULL DEFAULT 'PendingReview',
    "conflict_status" edi_transfer_change_conflict_status_enum NOT NULL DEFAULT 'None',
    "conflict_reason" text,
    "idempotency_key" varchar(255) NOT NULL,
    "source_shipment_version" bigint NOT NULL,
    "target_shipment_version" bigint NOT NULL,
    "payload" jsonb NOT NULL DEFAULT '{}',
    "diff" jsonb NOT NULL DEFAULT '{}',
    "reviewed_by_id" varchar(100),
    "reviewed_at" bigint,
    "applied_by_id" varchar(100),
    "applied_at" bigint,
    "failure_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_transfer_changes" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_transfer_changes_link" FOREIGN KEY ("shipment_link_id", "business_unit_id") REFERENCES "edi_shipment_links"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_transfer_changes_reviewed_by" FOREIGN KEY ("reviewed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_transfer_changes_applied_by" FOREIGN KEY ("applied_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transfer_changes_idempotency"
    ON "edi_transfer_changes"("shipment_link_id", "business_unit_id", "direction", "change_type", "idempotency_key");

CREATE INDEX IF NOT EXISTS "idx_edi_transfer_changes_link_status"
    ON "edi_transfer_changes"("shipment_link_id", "business_unit_id", "status", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_edi_transfer_changes_conflict"
    ON "edi_transfer_changes"("business_unit_id", "conflict_status", "status", "created_at" DESC);

--bun:split
ALTER TABLE "edi_transfer_changes"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("change_type", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("direction"), '')), 'C') ||
        setweight(immutable_to_tsvector('english', COALESCE("conflict_reason", '')), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS "idx_edi_transfer_changes_search"
    ON "edi_transfer_changes" USING GIN("search_vector");
