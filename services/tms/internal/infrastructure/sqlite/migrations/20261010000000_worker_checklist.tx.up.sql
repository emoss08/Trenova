-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261010000000_worker_checklist.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_checklist_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL DEFAULT 'Custom',
    "trigger" TEXT NOT NULL DEFAULT 'Manual',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_checklist_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklist_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_checklist_templates_code" ON "worker_checklist_templates" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_checklist_templates_default_trigger" ON "worker_checklist_templates" ("organization_id", "business_unit_id", "trigger")WHERE
    "is_default" AND "trigger" <> 'Manual';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_checklist_templates_status" ON "worker_checklist_templates" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_checklist_template_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL DEFAULT 'Task',
    "required" INTEGER NOT NULL DEFAULT 1,
    "due_offset_days" INTEGER NOT NULL DEFAULT 0,
    "owner" TEXT NOT NULL DEFAULT 'HR',
    "credential_type_id" TEXT,
    "document_type_id" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_checklist_template_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_checklist_template_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "worker_checklist_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_checklist_template_items_credential_type" FOREIGN KEY ("credential_type_id", "organization_id", "business_unit_id") REFERENCES "worker_credential_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_checklist_template_items_document_type" FOREIGN KEY ("document_type_id", "business_unit_id", "organization_id") REFERENCES "document_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_checklist_template_items_due" CHECK ("due_offset_days" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_checklist_template_items_template" ON "worker_checklist_template_items" ("template_id", "sort_order");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_checklists"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "template_id" TEXT,
    "name" TEXT NOT NULL,
    "kind" TEXT NOT NULL DEFAULT 'Custom',
    "status" TEXT NOT NULL DEFAULT 'Open',
    "started_at" INTEGER NOT NULL,
    "due_at" INTEGER,
    "completed_at" INTEGER,
    "cancelled_at" INTEGER,
    "cancel_reason" TEXT,
    "source_event_id" TEXT,
    "started_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE INDEX IF NOT EXISTS "idx_worker_checklists_worker" ON "worker_checklists" ("worker_id", "status", "started_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_checklists_open" ON "worker_checklists" ("organization_id", "business_unit_id", "due_at")WHERE
    "status" = 'Open';

--bun:split

CREATE TABLE IF NOT EXISTS "worker_checklist_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "checklist_id" TEXT NOT NULL,
    "template_item_id" TEXT,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL DEFAULT 'Task',
    "required" INTEGER NOT NULL DEFAULT 1,
    "owner" TEXT NOT NULL DEFAULT 'HR',
    "due_at" INTEGER,
    "credential_type_id" TEXT,
    "document_type_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "completed_by_id" TEXT,
    "completed_at" INTEGER,
    "auto_completed" INTEGER NOT NULL DEFAULT 0,
    "note" TEXT,
    "evidence_document_id" TEXT,
    "evidence_credential_id" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE INDEX IF NOT EXISTS "idx_worker_checklist_items_checklist" ON "worker_checklist_items" ("checklist_id", "sort_order");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_checklist_items_due" ON "worker_checklist_items" ("organization_id", "business_unit_id", "due_at")WHERE
    "status" = 'Pending' AND "due_at" IS NOT NULL;
