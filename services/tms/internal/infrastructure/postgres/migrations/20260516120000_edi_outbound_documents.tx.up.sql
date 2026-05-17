CREATE TYPE edi_document_direction_enum AS ENUM(
    'Inbound',
    'Outbound'
);

CREATE TYPE edi_standard_enum AS ENUM(
    'X12'
);

CREATE TYPE edi_transaction_set_enum AS ENUM(
    '204',
    '210',
    '214',
    '990',
    '997',
    '999'
);

CREATE TYPE edi_document_status_enum AS ENUM(
    'Active',
    'Inactive'
);

CREATE TYPE edi_template_status_enum AS ENUM(
    'Draft',
    'Certified',
    'Active',
    'Deprecated',
    'Archived',
    'Superseded'
);

CREATE TYPE edi_message_status_enum AS ENUM(
    'Generated',
    'Failed'
);

CREATE TYPE edi_validation_mode_enum AS ENUM(
    'Strict',
    'WarnOnly',
    'Disabled'
);

CREATE TYPE edi_validation_severity_enum AS ENUM(
    'Info',
    'Warning',
    'Error'
);

CREATE TYPE edi_control_number_kind_enum AS ENUM(
    'Interchange',
    'Group',
    'Transaction'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_transaction_sets"(
    "id" varchar(100) NOT NULL,
    "standard" edi_standard_enum NOT NULL,
    "code" varchar(20) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "default_version" varchar(20) NOT NULL DEFAULT '004010',
    "status" edi_document_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_transaction_sets" PRIMARY KEY ("id"),
    CONSTRAINT "uq_edi_transaction_sets_standard_code" UNIQUE ("standard", "code")
);

INSERT INTO "edi_transaction_sets"("id", "standard", "code", "name", "description", "default_version", "status")
VALUES
    ('edits_x12_204', 'X12', '204', 'Motor Carrier Load Tender', 'Outbound load tender document.', '004010', 'Active'),
    ('edits_x12_990', 'X12', '990', 'Response to a Load Tender', 'Load tender acceptance or rejection.', '004010', 'Active'),
    ('edits_x12_214', 'X12', '214', 'Transportation Carrier Shipment Status Message', 'Shipment status updates.', '004010', 'Active'),
    ('edits_x12_210', 'X12', '210', 'Motor Carrier Freight Details and Invoice', 'Carrier freight invoice.', '004010', 'Active'),
    ('edits_x12_997', 'X12', '997', 'Functional Acknowledgment', 'Functional acknowledgment.', '004010', 'Active'),
    ('edits_x12_999', 'X12', '999', 'Implementation Acknowledgment', 'Implementation acknowledgment.', '005010', 'Active')
ON CONFLICT ("standard", "code") DO NOTHING;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_document_types"(
    "id" varchar(100) NOT NULL,
    "code" varchar(20) NOT NULL,
    "name" varchar(200) NOT NULL,
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "transaction_set_id" varchar(100) NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "default_version" varchar(20) NOT NULL DEFAULT '004010',
    "status" edi_document_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_document_types" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_document_types_transaction_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "uq_edi_document_types_code" UNIQUE ("code"),
    CONSTRAINT "uq_edi_document_types_catalog" UNIQUE ("standard", "transaction_set", "direction")
);

INSERT INTO "edi_document_types"("id", "code", "name", "standard", "transaction_set", "transaction_set_id", "direction", "default_version", "status")
VALUES
    ('edidt_x12_204_outbound', 'X12-204-OUT', 'X12 204 Motor Carrier Load Tender', 'X12', '204', 'edits_x12_204', 'Outbound', '004010', 'Active'),
    ('edidt_x12_990_inbound', 'X12-990-IN', 'X12 990 Response to Load Tender', 'X12', '990', 'edits_x12_990', 'Inbound', '004010', 'Active'),
    ('edidt_x12_214_inbound', 'X12-214-IN', 'X12 214 Shipment Status', 'X12', '214', 'edits_x12_214', 'Inbound', '004010', 'Active'),
    ('edidt_x12_210_inbound', 'X12-210-IN', 'X12 210 Freight Invoice', 'X12', '210', 'edits_x12_210', 'Inbound', '004010', 'Active'),
    ('edidt_x12_997_inbound', 'X12-997-IN', 'X12 997 Functional Acknowledgment', 'X12', '997', 'edits_x12_997', 'Inbound', '004010', 'Active'),
    ('edidt_x12_999_inbound', 'X12-999-IN', 'X12 999 Implementation Acknowledgment', 'X12', '999', 'edits_x12_999', 'Inbound', '005010', 'Active')
ON CONFLICT ("code") DO NOTHING;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_transaction_loop_definitions"(
    "id" varchar(100) NOT NULL,
    "transaction_set_id" varchar(100) NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL DEFAULT '004010',
    "loop_id" varchar(50) NOT NULL,
    "name" varchar(200) NOT NULL,
    "parent_loop_id" varchar(50),
    "sequence" bigint NOT NULL,
    "repeat_path" text,
    "usage_notes" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_transaction_loop_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_loop_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_loop_definitions_scope"
    ON "edi_transaction_loop_definitions"("transaction_set_id", "direction", "x12_version", "loop_id");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_transaction_segment_definitions"(
    "id" varchar(100) NOT NULL,
    "transaction_set_id" varchar(100) NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL DEFAULT '004010',
    "segment_id" varchar(10) NOT NULL,
    "name" varchar(200) NOT NULL,
    "loop_id" varchar(50),
    "sequence" bigint NOT NULL,
    "required" boolean NOT NULL DEFAULT FALSE,
    "max_use" bigint NOT NULL DEFAULT 1,
    "repeat_path" text,
    "usage_notes" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_transaction_segment_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_segment_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_segment_definitions_scope"
    ON "edi_transaction_segment_definitions"("transaction_set_id", "direction", "x12_version", "segment_id", "sequence");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_code_list_definitions"(
    "id" varchar(100) NOT NULL,
    "transaction_set_id" varchar(100) NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL DEFAULT '004010',
    "element_id" varchar(20) NOT NULL,
    "code" varchar(40) NOT NULL,
    "description" text NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_code_list_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_code_list_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_code_list_definitions_scope"
    ON "edi_code_list_definitions"("transaction_set_id", "direction", "x12_version", "element_id", "code");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_transaction_element_definitions"(
    "id" varchar(100) NOT NULL,
    "transaction_set_id" varchar(100) NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL DEFAULT '004010',
    "segment_id" varchar(10) NOT NULL,
    "position" integer NOT NULL,
    "element_id" varchar(20),
    "name" varchar(200) NOT NULL,
    "required" boolean NOT NULL DEFAULT FALSE,
    "min_length" integer NOT NULL DEFAULT 0,
    "max_length" integer NOT NULL DEFAULT 0,
    "code_list_id" varchar(100),
    "usage_notes" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_transaction_element_definitions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_transaction_element_definitions_set" FOREIGN KEY ("transaction_set_id") REFERENCES "edi_transaction_sets"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_transaction_element_definitions_code_list" FOREIGN KEY ("code_list_id") REFERENCES "edi_code_list_definitions"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_transaction_element_definitions_scope"
    ON "edi_transaction_element_definitions"("transaction_set_id", "direction", "x12_version", "segment_id", "position");

INSERT INTO "edi_transaction_loop_definitions"("id", "transaction_set_id", "direction", "x12_version", "loop_id", "name", "sequence", "repeat_path")
VALUES
    ('ediloop_x12_204_out_root', 'edits_x12_204', 'Outbound', '004010', 'ROOT', 'Transaction root', 10, NULL),
    ('ediloop_x12_204_out_stop', 'edits_x12_204', 'Outbound', '004010', '0300', 'Stop-off details', 90, 'moves.0.stops'),
    ('ediloop_x12_204_out_commodity', 'edits_x12_204', 'Outbound', '004010', '0400', 'Commodity description', 150, 'commodities')
ON CONFLICT ("transaction_set_id", "direction", "x12_version", "loop_id") DO NOTHING;

INSERT INTO "edi_transaction_segment_definitions"("id", "transaction_set_id", "direction", "x12_version", "segment_id", "name", "loop_id", "sequence", "required", "max_use", "repeat_path")
VALUES
    ('edisegd_x12_204_out_isa', 'edits_x12_204', 'Outbound', '004010', 'ISA', 'Interchange Control Header', 'ROOT', 10, TRUE, 1, NULL),
    ('edisegd_x12_204_out_gs', 'edits_x12_204', 'Outbound', '004010', 'GS', 'Functional Group Header', 'ROOT', 20, TRUE, 1, NULL),
    ('edisegd_x12_204_out_st', 'edits_x12_204', 'Outbound', '004010', 'ST', 'Transaction Set Header', 'ROOT', 30, TRUE, 1, NULL),
    ('edisegd_x12_204_out_b2', 'edits_x12_204', 'Outbound', '004010', 'B2', 'Beginning Segment for Shipment Information', 'ROOT', 40, TRUE, 1, NULL),
    ('edisegd_x12_204_out_b2a', 'edits_x12_204', 'Outbound', '004010', 'B2A', 'Set Purpose', 'ROOT', 50, TRUE, 1, NULL),
    ('edisegd_x12_204_out_l11', 'edits_x12_204', 'Outbound', '004010', 'L11', 'Reference Identification', 'ROOT', 60, FALSE, 10, NULL),
    ('edisegd_x12_204_out_g62', 'edits_x12_204', 'Outbound', '004010', 'G62', 'Date Time', 'ROOT', 70, FALSE, 10, 'moves.0.stops'),
    ('edisegd_x12_204_out_nte', 'edits_x12_204', 'Outbound', '004010', 'NTE', 'Note', 'ROOT', 80, FALSE, 10, NULL),
    ('edisegd_x12_204_out_n1', 'edits_x12_204', 'Outbound', '004010', 'N1', 'Name', '0300', 90, FALSE, 1, 'moves.0.stops'),
    ('edisegd_x12_204_out_n3', 'edits_x12_204', 'Outbound', '004010', 'N3', 'Address', '0300', 100, FALSE, 2, 'moves.0.stops'),
    ('edisegd_x12_204_out_n4', 'edits_x12_204', 'Outbound', '004010', 'N4', 'Geographic Location', '0300', 110, FALSE, 1, 'moves.0.stops'),
    ('edisegd_x12_204_out_g61', 'edits_x12_204', 'Outbound', '004010', 'G61', 'Contact', '0300', 120, FALSE, 3, NULL),
    ('edisegd_x12_204_out_s5', 'edits_x12_204', 'Outbound', '004010', 'S5', 'Stop Off Details', '0300', 130, TRUE, 999, 'moves.0.stops'),
    ('edisegd_x12_204_out_at8', 'edits_x12_204', 'Outbound', '004010', 'AT8', 'Shipment Weight Packaging and Quantity Data', 'ROOT', 140, FALSE, 1, NULL),
    ('edisegd_x12_204_out_l5', 'edits_x12_204', 'Outbound', '004010', 'L5', 'Description Marks and Numbers', '0400', 150, FALSE, 999, 'commodities'),
    ('edisegd_x12_204_out_l3', 'edits_x12_204', 'Outbound', '004010', 'L3', 'Total Weight and Charges', 'ROOT', 160, FALSE, 1, NULL),
    ('edisegd_x12_204_out_se', 'edits_x12_204', 'Outbound', '004010', 'SE', 'Transaction Set Trailer', 'ROOT', 170, TRUE, 1, NULL),
    ('edisegd_x12_204_out_ge', 'edits_x12_204', 'Outbound', '004010', 'GE', 'Functional Group Trailer', 'ROOT', 180, TRUE, 1, NULL),
    ('edisegd_x12_204_out_iea', 'edits_x12_204', 'Outbound', '004010', 'IEA', 'Interchange Control Trailer', 'ROOT', 190, TRUE, 1, NULL)
ON CONFLICT ("transaction_set_id", "direction", "x12_version", "segment_id", "sequence") DO NOTHING;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "direction" edi_document_direction_enum NOT NULL,
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "status" edi_template_status_enum NOT NULL DEFAULT 'Active',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_templates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_templates_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_templates_name_org_bu"
    ON "edi_templates"(lower("name"), "business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_edi_templates_document_type"
    ON "edi_templates"("document_type_id", "business_unit_id", "organization_id", "status");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_template_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "source_version_id" varchar(100),
    "version_number" bigint NOT NULL,
    "x12_version" varchar(20) NOT NULL DEFAULT '004010',
    "functional_group_id" varchar(2) NOT NULL DEFAULT 'SM',
    "status" edi_template_status_enum NOT NULL DEFAULT 'Active',
    "is_active" boolean NOT NULL DEFAULT FALSE,
    "notes" text,
    "certification_notes" text,
    "activation_notes" text,
    "archive_notes" text,
    "deprecated_notes" text,
    "superseded_notes" text,
    "certified_by_id" varchar(100),
    "activated_by_id" varchar(100),
    "archived_by_id" varchar(100),
    "deprecated_by_id" varchar(100),
    "superseded_by_id" varchar(100),
    "certified_at" bigint,
    "activated_at" bigint,
    "archived_at" bigint,
    "deprecated_at" bigint,
    "superseded_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_template_versions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_versions_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_template_versions_source" FOREIGN KEY ("source_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("source_version_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_versions_number"
    ON "edi_template_versions"("template_id", "business_unit_id", "organization_id", "version_number");

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_versions_active"
    ON "edi_template_versions"("template_id", "business_unit_id", "organization_id")
    WHERE "is_active" = TRUE;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_template_segments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "template_version_id" varchar(100) NOT NULL,
    "segment_id" varchar(10) NOT NULL,
    "name" varchar(200) NOT NULL,
    "sequence" bigint NOT NULL,
    "loop_id" varchar(50),
    "repeat_path" text,
    "condition" text,
    "required" boolean NOT NULL DEFAULT FALSE,
    "max_use" bigint NOT NULL DEFAULT 1,
    "elements" jsonb NOT NULL DEFAULT '[]',
    "usage_notes" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_template_segments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_segments_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_segments_sequence"
    ON "edi_template_segments"("template_version_id", "business_unit_id", "organization_id", "sequence");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_partner_document_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "template_version_id" varchar(100),
    "name" varchar(200) NOT NULL,
    "status" edi_document_status_enum NOT NULL DEFAULT 'Active',
    "direction" edi_document_direction_enum NOT NULL,
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "x12_version_override" varchar(20),
    "functional_group_id" varchar(2) NOT NULL DEFAULT 'SM',
    "envelope" jsonb NOT NULL DEFAULT '{}',
    "acknowledgment" jsonb NOT NULL DEFAULT '{}',
    "validation_mode" edi_validation_mode_enum NOT NULL DEFAULT 'Strict',
    "partner_settings" jsonb NOT NULL DEFAULT '{}',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_partner_document_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_partner_document_profiles_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partner_document_profiles_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_partner_document_profiles_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_partner_document_profiles_template_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("template_version_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_document_profiles_active"
    ON "edi_partner_document_profiles"("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "direction")
    WHERE "status" = 'Active';

--bun:split
CREATE TABLE IF NOT EXISTS "edi_control_number_sequences"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "kind" edi_control_number_kind_enum NOT NULL,
    "next_value" bigint NOT NULL DEFAULT 1,
    "min_value" bigint NOT NULL DEFAULT 1,
    "max_value" bigint NOT NULL DEFAULT 999999999,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_control_number_sequences" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_control_number_sequences_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_control_number_sequences_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "ck_edi_control_number_sequences_range" CHECK ("min_value" > 0 AND "max_value" >= "min_value" AND "next_value" >= "min_value")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_control_number_sequences_scope"
    ON "edi_control_number_sequences"("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "kind");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_messages"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "partner_document_profile_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "template_version_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100),
    "transfer_id" varchar(100),
    "direction" edi_document_direction_enum NOT NULL,
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL,
    "status" edi_message_status_enum NOT NULL,
    "validation_mode" edi_validation_mode_enum NOT NULL,
    "interchange_control_number" varchar(20) NOT NULL,
    "group_control_number" varchar(20) NOT NULL,
    "transaction_control_number" varchar(20) NOT NULL,
    "segment_count" bigint NOT NULL,
    "raw_x12" text NOT NULL,
    "payload_snapshot" jsonb NOT NULL,
    "generated_by_id" varchar(100),
    "generated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_messages_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_template" FOREIGN KEY ("template_id", "business_unit_id", "organization_id") REFERENCES "edi_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_template_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_edi_messages_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_messages_control_numbers"
    ON "edi_messages"("business_unit_id", "organization_id", "edi_partner_id", "document_type_id", "interchange_control_number", "group_control_number", "transaction_control_number");

CREATE INDEX IF NOT EXISTS "idx_edi_messages_archive"
    ON "edi_messages"("business_unit_id", "organization_id", "transaction_set", "direction", "generated_at" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_message_validation_errors"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "message_id" varchar(100) NOT NULL,
    "severity" edi_validation_severity_enum NOT NULL,
    "code" varchar(100) NOT NULL,
    "segment_id" varchar(10),
    "element_position" integer NOT NULL DEFAULT 0,
    "path" text,
    "message" text NOT NULL,
    "suggested_fix" text,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_message_validation_errors" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_message_validation_errors_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "edi_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_edi_message_validation_errors_message"
    ON "edi_message_validation_errors"("message_id", "business_unit_id", "organization_id", "severity");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_test_cases"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "partner_document_profile_id" varchar(100) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "payload" jsonb NOT NULL,
    "expected_warnings" integer NOT NULL DEFAULT 0,
    "expected_errors" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_test_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_test_cases_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_test_cases_name_profile"
    ON "edi_test_cases"("partner_document_profile_id", "business_unit_id", "organization_id", lower("name"));
