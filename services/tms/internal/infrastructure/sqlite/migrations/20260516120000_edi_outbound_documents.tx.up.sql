-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260516120000_edi_outbound_documents.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_transaction_sets"(
    "id" TEXT NOT NULL,
    "standard" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "default_version" TEXT NOT NULL DEFAULT '004010',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_transaction_sets" PRIMARY KEY ("id"),
    CONSTRAINT "uq_edi_transaction_sets_standard_code" UNIQUE ("standard", "code")
);

--bun:split

CREATE TABLE IF NOT EXISTS "edi_document_types"(
    "id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "transaction_set_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "default_version" TEXT NOT NULL DEFAULT '004010',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_document_types" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_document_types_transaction_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "uq_edi_document_types_code" UNIQUE ("code"),
    CONSTRAINT "uq_edi_document_types_catalog" UNIQUE ("standard", "transaction_set", "direction")
);

--bun:split

CREATE TABLE IF NOT EXISTS "edi_transaction_loop_definitions"(
    "id" TEXT NOT NULL,
    "transaction_set_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL DEFAULT '004010',
    "loop_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "parent_loop_id" TEXT,
    "sequence" INTEGER NOT NULL,
    "repeat_path" TEXT,
    "usage_notes" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_transaction_loop_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_loop_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_loop_definitions_scope"
    ON "edi_transaction_loop_definitions" ("transaction_set_id", "direction", "x12_version", "loop_id");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_transaction_segment_definitions"(
    "id" TEXT NOT NULL,
    "transaction_set_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL DEFAULT '004010',
    "segment_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "loop_id" TEXT,
    "sequence" INTEGER NOT NULL,
    "required" INTEGER NOT NULL DEFAULT 0,
    "max_use" INTEGER NOT NULL DEFAULT 1,
    "repeat_path" TEXT,
    "usage_notes" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_transaction_segment_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_segment_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_segment_definitions_scope"
    ON "edi_transaction_segment_definitions" ("transaction_set_id", "direction", "x12_version", "segment_id", "sequence");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_code_list_definitions"(
    "id" TEXT NOT NULL,
    "transaction_set_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL DEFAULT '004010',
    "element_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_code_list_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_code_list_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_code_list_definitions_scope"
    ON "edi_code_list_definitions" ("transaction_set_id", "direction", "x12_version", "element_id", "code");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_transaction_element_definitions"(
    "id" TEXT NOT NULL,
    "transaction_set_id" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL DEFAULT '004010',
    "segment_id" TEXT NOT NULL,
    "position" INTEGER NOT NULL,
    "element_id" TEXT,
    "name" TEXT NOT NULL,
    "required" INTEGER NOT NULL DEFAULT 0,
    "min_length" INTEGER NOT NULL DEFAULT 0,
    "max_length" INTEGER NOT NULL DEFAULT 0,
    "code_list_id" TEXT,
    "usage_notes" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_transaction_element_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_element_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_transaction_element_definitions_code_list" FOREIGN KEY ("code_list_id") REFERENCES "edi_code_list_definitions"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_element_definitions_scope"
    ON "edi_transaction_element_definitions" ("transaction_set_id", "direction", "x12_version", "segment_id", "position");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "direction" TEXT NOT NULL,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_templates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_templates_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_templates_name_org_bu"
    ON "edi_templates" (lower("name"), "business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_templates_document_type"
    ON "edi_templates" ("document_type_id", "business_unit_id", "organization_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_template_versions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "source_version_id" TEXT,
    "version_number" INTEGER NOT NULL,
    "x12_version" TEXT NOT NULL DEFAULT '004010',
    "functional_group_id" TEXT NOT NULL DEFAULT 'SM',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_active" INTEGER NOT NULL DEFAULT 0,
    "notes" TEXT,
    "certification_notes" TEXT,
    "activation_notes" TEXT,
    "archive_notes" TEXT,
    "deprecated_notes" TEXT,
    "superseded_notes" TEXT,
    "certified_by_id" TEXT,
    "activated_by_id" TEXT,
    "archived_by_id" TEXT,
    "deprecated_by_id" TEXT,
    "superseded_by_id" TEXT,
    "certified_at" INTEGER,
    "activated_at" INTEGER,
    "archived_at" INTEGER,
    "deprecated_at" INTEGER,
    "superseded_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_template_versions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_versions_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_template_versions_source" FOREIGN KEY ("source_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_versions_number"
    ON "edi_template_versions" ("template_id", "business_unit_id", "organization_id", "version_number");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_versions_active"
    ON "edi_template_versions" ("template_id", "business_unit_id", "organization_id")WHERE "is_active" = TRUE;

--bun:split

CREATE TABLE IF NOT EXISTS "edi_template_segments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "template_version_id" TEXT NOT NULL,
    "segment_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL,
    "loop_id" TEXT,
    "repeat_path" TEXT,
    "condition" TEXT,
    "required" INTEGER NOT NULL DEFAULT 0,
    "max_use" INTEGER NOT NULL DEFAULT 1,
    "elements" TEXT NOT NULL DEFAULT '[]',
    "usage_notes" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_template_segments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_segments_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_segments_sequence"
    ON "edi_template_segments" ("template_version_id", "business_unit_id", "organization_id", "sequence");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_partner_document_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "template_version_id" TEXT,
    "name" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "direction" TEXT NOT NULL,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "x12_version_override" TEXT,
    "functional_group_id" TEXT NOT NULL DEFAULT 'SM',
    "envelope" TEXT NOT NULL DEFAULT '{}',
    "acknowledgment" TEXT NOT NULL DEFAULT '{}',
    "validation_mode" TEXT NOT NULL DEFAULT 'Strict',
    "partner_settings" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_partner_document_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_partner_document_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partner_document_profiles_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_partner_document_profiles_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_partner_document_profiles_template_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_document_profiles_active"
    ON "edi_partner_document_profiles" ("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "direction")WHERE "status" = 'Active';

--bun:split

CREATE TABLE IF NOT EXISTS "edi_control_number_sequences"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "next_value" INTEGER NOT NULL DEFAULT 1,
    "min_value" INTEGER NOT NULL DEFAULT 1,
    "max_value" INTEGER NOT NULL DEFAULT 999999999,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_control_number_sequences" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_control_number_sequences_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_control_number_sequences_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "ck_edi_control_number_sequences_range" CHECK ("min_value" > 0 AND "max_value" >= "min_value" AND "next_value" >= "min_value")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_control_number_sequences_scope"
    ON "edi_control_number_sequences" ("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "kind");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_messages"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    "partner_document_profile_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "template_version_id" TEXT NOT NULL,
    "shipment_id" TEXT,
    "transfer_id" TEXT,
    "direction" TEXT NOT NULL,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "validation_mode" TEXT NOT NULL,
    "interchange_control_number" TEXT NOT NULL,
    "group_control_number" TEXT NOT NULL,
    "transaction_control_number" TEXT NOT NULL,
    "segment_count" INTEGER NOT NULL,
    "raw_x12" TEXT NOT NULL,
    "payload_snapshot" TEXT NOT NULL,
    "generated_by_id" TEXT,
    "generated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_messages_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_template_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_messages_control_numbers"
    ON "edi_messages" ("business_unit_id", "organization_id", "edi_partner_id", "document_type_id", "interchange_control_number", "group_control_number", "transaction_control_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_archive"
    ON "edi_messages" ("business_unit_id", "organization_id", "transaction_set", "direction", "generated_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "edi_message_validation_errors"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT NOT NULL,
    "severity" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "segment_id" TEXT,
    "element_position" INTEGER NOT NULL DEFAULT 0,
    "path" TEXT,
    "message" TEXT NOT NULL,
    "suggested_fix" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_message_validation_errors" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_message_validation_errors_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "edi_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_message_validation_errors_message"
    ON "edi_message_validation_errors" ("message_id", "business_unit_id", "organization_id", "severity");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_test_cases"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "partner_document_profile_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "payload" TEXT NOT NULL,
    "expected_warnings" INTEGER NOT NULL DEFAULT 0,
    "expected_errors" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_test_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_test_cases_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_test_cases_name_profile"
    ON "edi_test_cases" ("partner_document_profile_id", "business_unit_id", "organization_id", lower("name"));
