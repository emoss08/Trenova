-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261024000000_self_service.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_policies"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "summary" TEXT,
    "body" TEXT,
    "document_id" TEXT,
    "version_label" TEXT NOT NULL DEFAULT '1',
    "requires_signature" INTEGER NOT NULL DEFAULT 1,
    "applies_to" TEXT NOT NULL DEFAULT 'All',
    "effective_from" INTEGER NOT NULL,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policies_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_policies_code" ON "worker_policies" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_policies_active" ON "worker_policies" ("organization_id", "business_unit_id", "status", "effective_from");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_policy_acknowledgements"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "policy_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "version_label" TEXT NOT NULL,
    "acknowledged_at" INTEGER NOT NULL,
    "signature_name" TEXT,
    "signature_ip" TEXT,
    "signature_user_agent" TEXT,
    "document_checksum" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_policy_acknowledgements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_policy_acknowledgements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_policy" FOREIGN KEY ("policy_id", "organization_id", "business_unit_id") REFERENCES "worker_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_policy_acknowledgements_version" ON "worker_policy_acknowledgements" ("organization_id", "business_unit_id", "policy_id", "worker_id", "version_label");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_policy_acknowledgements_worker" ON "worker_policy_acknowledgements" ("organization_id", "business_unit_id", "worker_id", "acknowledged_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_profile_change_requests"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "changes" TEXT NOT NULL,
    "note" TEXT,
    "submitted_at" INTEGER NOT NULL,
    "decided_at" INTEGER,
    "decided_by_id" TEXT,
    "decision_note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_profile_change_requests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_profile_change_requests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profile_change_requests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profile_change_requests_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_profile_change_requests_open" ON "worker_profile_change_requests" ("organization_id", "business_unit_id", "worker_id")WHERE
    "status" = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profile_change_requests_queue" ON "worker_profile_change_requests" ("organization_id", "business_unit_id", "status", "submitted_at" DESC);

--bun:split

ALTER TABLE "dash_controls" ADD COLUMN "require_contact_change_approval" INTEGER NOT NULL DEFAULT 0;
