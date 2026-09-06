--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "worker_checklist_kind_enum" AS ENUM(
    'Onboarding',
    'Offboarding',
    'Custom'
);

CREATE TYPE "worker_checklist_trigger_enum" AS ENUM(
    'Hired',
    'Rehired',
    'Terminated',
    'Manual'
);

CREATE TYPE "worker_checklist_item_kind_enum" AS ENUM(
    'Document',
    'Credential',
    'Task',
    'Equipment',
    'PortalAccess'
);

CREATE TYPE "worker_checklist_owner_enum" AS ENUM(
    'HR',
    'Safety',
    'Dispatch',
    'Payroll',
    'IT',
    'Fleet'
);

CREATE TYPE "worker_checklist_status_enum" AS ENUM(
    'Open',
    'Completed',
    'Cancelled'
);

CREATE TYPE "worker_checklist_item_status_enum" AS ENUM(
    'Pending',
    'Done',
    'Skipped',
    'NotApplicable'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_checklist_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "kind" worker_checklist_kind_enum NOT NULL DEFAULT 'Custom',
    "trigger" worker_checklist_trigger_enum NOT NULL DEFAULT 'Manual',
    "status" status_enum NOT NULL DEFAULT 'Active',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_checklist_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklist_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
--bun:split
ALTER TABLE "worker_checklist_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_checklist_templates_code" ON "worker_checklist_templates"("organization_id", "business_unit_id", LOWER("code"));
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_checklist_templates_default_trigger" ON "worker_checklist_templates"("organization_id", "business_unit_id", "trigger")
WHERE
    "is_default" AND "trigger" <> 'Manual';
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklist_templates_status" ON "worker_checklist_templates"("organization_id", "business_unit_id", "status");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklist_templates_search" ON "worker_checklist_templates" USING GIN(search_vector);
--bun:split
COMMENT ON TABLE worker_checklist_templates IS 'Reusable onboarding / offboarding checklists. At most one default template per trigger spawns automatically from the matching employment event.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_checklist_template_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "label" varchar(150) NOT NULL,
    "description" text,
    "kind" worker_checklist_item_kind_enum NOT NULL DEFAULT 'Task',
    "required" boolean NOT NULL DEFAULT TRUE,
    "due_offset_days" integer NOT NULL DEFAULT 0,
    "owner" worker_checklist_owner_enum NOT NULL DEFAULT 'HR',
    "credential_type_id" varchar(100),
    "document_type_id" varchar(100),
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_checklist_template_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklist_template_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "worker_checklist_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_credential_type" FOREIGN KEY ("credential_type_id", "organization_id", "business_unit_id") REFERENCES "worker_credential_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklist_template_items_document_type" FOREIGN KEY ("document_type_id", "business_unit_id", "organization_id") REFERENCES "document_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_checklist_template_items_due" CHECK ("due_offset_days" >= 0)
);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklist_template_items_template" ON "worker_checklist_template_items"("template_id", "sort_order");
--bun:split
CREATE TABLE IF NOT EXISTS "worker_checklists"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "template_id" varchar(100),
    "name" varchar(100) NOT NULL,
    "kind" worker_checklist_kind_enum NOT NULL DEFAULT 'Custom',
    "status" worker_checklist_status_enum NOT NULL DEFAULT 'Open',
    "started_at" bigint NOT NULL,
    "due_at" bigint,
    "completed_at" bigint,
    "cancelled_at" bigint,
    "cancel_reason" varchar(255),
    "source_event_id" varchar(100),
    "started_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_checklists" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklists_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklists_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklists_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklists_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "worker_checklist_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklists_source_event" FOREIGN KEY ("source_event_id", "organization_id", "business_unit_id") REFERENCES "worker_employment_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_checklists_completed" CHECK (("status" = 'Completed') = ("completed_at" IS NOT NULL)),
    CONSTRAINT "chk_worker_checklists_cancelled" CHECK (("status" = 'Cancelled') = ("cancelled_at" IS NOT NULL))
);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklists_worker" ON "worker_checklists"("worker_id", "status", "started_at" DESC);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklists_open" ON "worker_checklists"("organization_id", "business_unit_id", "due_at")
WHERE
    "status" = 'Open';
--bun:split
COMMENT ON TABLE worker_checklists IS 'A checklist instance spawned for one worker. Items are copied from the template so later template edits do not rewrite history.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_checklist_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "checklist_id" varchar(100) NOT NULL,
    "template_item_id" varchar(100),
    "label" varchar(150) NOT NULL,
    "description" text,
    "kind" worker_checklist_item_kind_enum NOT NULL DEFAULT 'Task',
    "required" boolean NOT NULL DEFAULT TRUE,
    "owner" worker_checklist_owner_enum NOT NULL DEFAULT 'HR',
    "due_at" bigint,
    "credential_type_id" varchar(100),
    "document_type_id" varchar(100),
    "status" worker_checklist_item_status_enum NOT NULL DEFAULT 'Pending',
    "completed_by_id" varchar(100),
    "completed_at" bigint,
    "auto_completed" boolean NOT NULL DEFAULT FALSE,
    "note" varchar(500),
    "evidence_document_id" varchar(100),
    "evidence_credential_id" varchar(100),
    "sort_order" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_checklist_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklist_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_items_checklist" FOREIGN KEY ("checklist_id", "organization_id", "business_unit_id") REFERENCES "worker_checklists"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_items_credential_type" FOREIGN KEY ("credential_type_id", "organization_id", "business_unit_id") REFERENCES "worker_credential_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklist_items_document_type" FOREIGN KEY ("document_type_id", "business_unit_id", "organization_id") REFERENCES "document_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklist_items_evidence_document" FOREIGN KEY ("evidence_document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklist_items_evidence_credential" FOREIGN KEY ("evidence_credential_id", "organization_id", "business_unit_id") REFERENCES "worker_credentials"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_checklist_items_completed" CHECK (("status" = 'Pending') = ("completed_at" IS NULL))
);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklist_items_checklist" ON "worker_checklist_items"("checklist_id", "sort_order");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_checklist_items_due" ON "worker_checklist_items"("organization_id", "business_unit_id", "due_at")
WHERE
    "status" = 'Pending' AND "due_at" IS NOT NULL;
--bun:split
COMMENT ON TABLE worker_checklist_items IS 'Items on a worker checklist. Credential, Document and PortalAccess items complete themselves when the evidence exists; Task and Equipment items are ticked by a person.';
