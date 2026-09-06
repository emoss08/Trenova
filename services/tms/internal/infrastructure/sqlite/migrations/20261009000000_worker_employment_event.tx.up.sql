-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261009000000_worker_employment_event.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_employment_events"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "effective_at" INTEGER NOT NULL,
    "reason" TEXT,
    "notes" TEXT,
    "from_values" TEXT NOT NULL DEFAULT '{}',
    "to_values" TEXT NOT NULL DEFAULT '{}',
    "document_id" TEXT,
    "recorded_by_id" TEXT,
    "amended_by_id" TEXT,
    "amended_at" INTEGER,
    "amendment_note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_employment_events" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_employment_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_employment_events_effective" CHECK ("effective_at" > 0),
    CONSTRAINT "chk_worker_employment_events_amendment" CHECK (("amended_at" IS NULL) = ("amended_by_id" IS NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_employment_events_worker" ON "worker_employment_events" ("worker_id", "effective_at" DESC, "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_employment_events_kind" ON "worker_employment_events" ("organization_id", "business_unit_id", "kind", "effective_at" DESC);
