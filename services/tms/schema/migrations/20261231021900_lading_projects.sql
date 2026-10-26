-- Lading replaces EDI templates, Starlark scripts, source contexts, partner setting schemas,
-- test cases and document profiles: a partner's EDI is one project of Lading files, saved as
-- versions (docs/engineering/lading.md). Nothing has launched, so the old records go with
-- their tables.
CREATE TYPE edi_lading_version_status_enum AS ENUM (
    'Draft',
    'Certified',
    'Active',
    'Shadow',
    'Superseded',
    'Archived'
);

CREATE TYPE edi_flow_instance_status_enum AS ENUM (
    'Open',
    'Held',
    'Escalated',
    'Closed'
);

CREATE TYPE edi_lading_replay_status_enum AS ENUM (
    'Running',
    'Completed',
    'Failed'
);

CREATE TABLE "edi_lading_projects"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "active_version_id" varchar(100),
    "shadow_version_id" varchar(100),
    "settings" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "next_version_number" bigint NOT NULL DEFAULT 1,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_lading_projects" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_lading_projects_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_lading_projects_settings" CHECK (jsonb_typeof("settings") = 'object')
);

CREATE UNIQUE INDEX "uq_edi_lading_projects_partner"
    ON "edi_lading_projects"("edi_partner_id", "business_unit_id", "organization_id");

CREATE TABLE "edi_lading_samples"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "project_id" varchar(100) NOT NULL,
    "digest" varchar(64) NOT NULL,
    "size" integer NOT NULL,
    "content" text NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_lading_samples" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_lading_samples_project" FOREIGN KEY ("project_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_projects"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_lading_samples_size" CHECK ("size" >= 0 AND "size" <= 1048576)
);

CREATE UNIQUE INDEX "uq_edi_lading_samples_digest"
    ON "edi_lading_samples"("project_id", "business_unit_id", "organization_id", "digest");

CREATE TABLE "edi_lading_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "project_id" varchar(100) NOT NULL,
    "number" bigint NOT NULL,
    "status" edi_lading_version_status_enum NOT NULL DEFAULT 'Draft',
    "parent_version_id" varchar(100),
    "files" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "samples" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "manifest" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "readiness" jsonb,
    "notes" text,
    "error_count" integer NOT NULL DEFAULT 0,
    "warning_count" integer NOT NULL DEFAULT 0,
    "shadow_until" bigint,
    "created_by_id" varchar(100),
    "certified_by_id" varchar(100),
    "certified_at" bigint,
    "activated_by_id" varchar(100),
    "activated_at" bigint,
    "superseded_at" bigint,
    "archived_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_lading_versions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_lading_versions_project" FOREIGN KEY ("project_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_projects"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_lading_versions_files" CHECK (jsonb_typeof("files") = 'array' AND jsonb_array_length("files") <= 200),
    CONSTRAINT "ck_edi_lading_versions_samples" CHECK (jsonb_typeof("samples") = 'array' AND jsonb_array_length("samples") <= 2000),
    CONSTRAINT "ck_edi_lading_versions_number" CHECK ("number" > 0)
);

CREATE UNIQUE INDEX "uq_edi_lading_versions_number"
    ON "edi_lading_versions"("project_id", "business_unit_id", "organization_id", "number");

CREATE UNIQUE INDEX "uq_edi_lading_versions_active"
    ON "edi_lading_versions"("project_id", "business_unit_id", "organization_id")
    WHERE "status" = 'Active';

CREATE INDEX "idx_edi_lading_versions_project"
    ON "edi_lading_versions"("project_id", "business_unit_id", "organization_id", "number" DESC);

ALTER TABLE "edi_lading_projects"
    ADD CONSTRAINT "fk_edi_lading_projects_active_version" FOREIGN KEY ("active_version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("active_version_id"),
    ADD CONSTRAINT "fk_edi_lading_projects_shadow_version" FOREIGN KEY ("shadow_version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("shadow_version_id");

CREATE TABLE "edi_flow_instances"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "project_id" varchar(100),
    "version_id" varchar(100),
    "flow" varchar(100) NOT NULL,
    "key" varchar(300) NOT NULL,
    "state" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "status" edi_flow_instance_status_enum NOT NULL DEFAULT 'Open',
    "reason" jsonb,
    "wake_at" bigint,
    "shipment_id" varchar(100),
    "transfer_id" varchar(100),
    "last_event_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "held_at" bigint,
    "resolved_by_id" varchar(100),
    "resolved_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_flow_instances" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_flow_instances_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_flow_instances_project" FOREIGN KEY ("project_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_projects"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("project_id"),
    CONSTRAINT "fk_edi_flow_instances_version" FOREIGN KEY ("version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("version_id"),
    CONSTRAINT "fk_edi_flow_instances_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("shipment_id"),
    CONSTRAINT "ck_edi_flow_instances_state" CHECK (jsonb_typeof("state") = 'object')
);

CREATE UNIQUE INDEX "uq_edi_flow_instances_key"
    ON "edi_flow_instances"("organization_id", "business_unit_id", "edi_partner_id", "flow", "key");

CREATE INDEX "idx_edi_flow_instances_queue"
    ON "edi_flow_instances"("organization_id", "business_unit_id", "status", "updated_at" DESC)
    WHERE "status" IN ('Held', 'Escalated');

CREATE INDEX "idx_edi_flow_instances_partner"
    ON "edi_flow_instances"("organization_id", "business_unit_id", "edi_partner_id", "last_event_at" DESC);

CREATE INDEX "idx_edi_flow_instances_shipment"
    ON "edi_flow_instances"("shipment_id", "business_unit_id", "organization_id")
    WHERE "shipment_id" IS NOT NULL;

CREATE INDEX "idx_edi_flow_instances_transfer"
    ON "edi_flow_instances"("transfer_id", "business_unit_id", "organization_id")
    WHERE "transfer_id" IS NOT NULL;

CREATE INDEX "idx_edi_flow_instances_wake"
    ON "edi_flow_instances"("wake_at")
    WHERE "wake_at" IS NOT NULL AND "status" <> 'Closed';

CREATE TABLE "edi_flow_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "instance_id" varchar(100) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "detail" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "actions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "error" jsonb,
    "message_id" varchar(100),
    "actor_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_flow_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_flow_events_instance" FOREIGN KEY ("instance_id", "business_unit_id", "organization_id") REFERENCES "edi_flow_instances"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX "idx_edi_flow_events_instance"
    ON "edi_flow_events"("instance_id", "business_unit_id", "organization_id", "created_at" DESC);

CREATE TABLE "edi_lading_replays"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "project_id" varchar(100) NOT NULL,
    "version_id" varchar(100) NOT NULL,
    "against_version_id" varchar(100) NOT NULL,
    "map" varchar(100) NOT NULL,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "status" edi_lading_replay_status_enum NOT NULL DEFAULT 'Running',
    "summary" jsonb,
    "items" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "error" text,
    "created_by_id" varchar(100),
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_lading_replays" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_lading_replays_project" FOREIGN KEY ("project_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_projects"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_lading_replays_version" FOREIGN KEY ("version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_lading_replays_against" FOREIGN KEY ("against_version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_edi_lading_replays_period" CHECK ("period_end" >= "period_start"),
    CONSTRAINT "ck_edi_lading_replays_items" CHECK (jsonb_typeof("items") = 'array')
);

CREATE INDEX "idx_edi_lading_replays_version"
    ON "edi_lading_replays"("version_id", "business_unit_id", "organization_id", "created_at" DESC);

CREATE TABLE "edi_lading_shadow_results"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "version_id" varchar(100) NOT NULL,
    "message_id" varchar(100) NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "same" boolean NOT NULL,
    "differences" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "new_outcomes" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "gone_outcomes" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "error" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_lading_shadow_results" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_lading_shadow_results_version" FOREIGN KEY ("version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_lading_shadow_results_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "edi_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX "uq_edi_lading_shadow_results_message"
    ON "edi_lading_shadow_results"("version_id", "business_unit_id", "organization_id", "message_id");

CREATE INDEX "idx_edi_lading_shadow_results_version"
    ON "edi_lading_shadow_results"("version_id", "business_unit_id", "organization_id", "created_at" DESC);

-- Messages record the version that read or wrote them, the flow instance they belong to, and
-- the document and lineage of the run.
ALTER TABLE "edi_messages"
    DROP COLUMN "partner_document_profile_id",
    DROP COLUMN "template_id",
    DROP COLUMN "template_version_id",
    DROP COLUMN "validation_mode",
    ADD COLUMN "lading_version_id" varchar(100),
    ADD COLUMN "flow_instance_id" varchar(100),
    ADD COLUMN "flow_send_id" varchar(100),
    ADD COLUMN "document" jsonb,
    ADD COLUMN "lineage" jsonb,
    ADD COLUMN "outcomes" jsonb,
    ADD CONSTRAINT "fk_edi_messages_lading_version" FOREIGN KEY ("lading_version_id", "business_unit_id", "organization_id") REFERENCES "edi_lading_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("lading_version_id"),
    ADD CONSTRAINT "fk_edi_messages_flow_instance" FOREIGN KEY ("flow_instance_id", "business_unit_id", "organization_id") REFERENCES "edi_flow_instances"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("flow_instance_id");

CREATE INDEX "idx_edi_messages_flow_send"
    ON "edi_messages"("flow_instance_id", "business_unit_id", "organization_id", "flow_send_id")
    WHERE "flow_instance_id" IS NOT NULL;

CREATE INDEX "idx_edi_messages_replay"
    ON "edi_messages"("organization_id", "business_unit_id", "edi_partner_id", "transaction_set", "direction", "generated_at");

ALTER TABLE "carrier_edi_channels"
    DROP COLUMN "partner_document_profile_id";

COMMENT ON TABLE "carrier_edi_channels" IS 'Links a carrier to the EDI partner and transport used to exchange tender documents with it';

DROP INDEX "idx_edi_tender_recipients_unique_recipient";

ALTER TABLE "edi_tender_recipients"
    DROP COLUMN "partner_document_profile_id";

CREATE UNIQUE INDEX "idx_edi_tender_recipients_unique_recipient"
    ON "edi_tender_recipients"(
        "source_shipment_id",
        "business_unit_id",
        "source_organization_id",
        "recipient_kind",
        COALESCE("recipient_organization_id", ''),
        COALESCE("edi_partner_id", '')
    );

DROP TABLE
    "edi_test_cases",
    "edi_partner_document_profiles",
    "edi_template_script_libraries",
    "edi_template_segments",
    "edi_template_versions",
    "edi_templates",
    "edi_source_context_fields",
    "edi_source_context_schemas",
    "edi_partner_setting_fields",
    "edi_partner_setting_schemas";

DROP TYPE
    "edi_template_status_enum",
    "edi_script_language_enum",
    "edi_partner_setting_data_type_enum",
    "edi_partner_setting_status_enum",
    "edi_source_context_data_type_enum",
    "edi_source_context_field_status_enum",
    "edi_source_context_kind_enum",
    "edi_validation_mode_enum";
