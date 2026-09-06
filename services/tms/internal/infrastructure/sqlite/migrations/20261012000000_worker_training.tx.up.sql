-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261012000000_worker_training.tx.up.sql

CREATE TABLE IF NOT EXISTS "training_courses"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT NOT NULL DEFAULT 'Other',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "delivery" TEXT NOT NULL DEFAULT 'Online',
    "content_url" TEXT,
    "duration_minutes" INTEGER NOT NULL DEFAULT 0,
    "passing_score" REAL,
    "validity_months" INTEGER,
    "renewal_window_days" INTEGER NOT NULL DEFAULT 30,
    "is_required" INTEGER NOT NULL DEFAULT 0,
    "required_for_driver_types" TEXT NOT NULL DEFAULT '[]',
    "due_days_after_assignment" INTEGER NOT NULL DEFAULT 30,
    "requires_acknowledgement" INTEGER NOT NULL DEFAULT 1,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_training_courses" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_training_courses_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_training_courses_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_training_courses_duration" CHECK ("duration_minutes" >= 0),
    CONSTRAINT "chk_training_courses_passing_score" CHECK ("passing_score" IS NULL OR ("passing_score" >= 0 AND "passing_score" <= 100)),
    CONSTRAINT "chk_training_courses_validity" CHECK ("validity_months" IS NULL OR "validity_months" > 0),
    CONSTRAINT "chk_training_courses_renewal_window" CHECK ("renewal_window_days" >= 0),
    CONSTRAINT "chk_training_courses_due_days" CHECK ("due_days_after_assignment" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_training_courses_code" ON "training_courses" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_training_courses_status" ON "training_courses" ("organization_id", "business_unit_id", "status", "sort_order");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_training_records"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "course_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Assigned',
    "assigned_at" INTEGER NOT NULL,
    "due_at" INTEGER,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "expires_at" INTEGER,
    "score" REAL,
    "passed" INTEGER,
    "acknowledged_at" INTEGER,
    "document_id" TEXT,
    "assigned_by_id" TEXT,
    "recorded_by_id" TEXT,
    "notes" TEXT,
    "waived_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_training_records" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_training_records_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_training_records_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_training_records_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_training_records_course" FOREIGN KEY ("course_id", "organization_id", "business_unit_id") REFERENCES "training_courses"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_worker_training_records_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_training_records_score" CHECK ("score" IS NULL OR ("score" >= 0 AND "score" <= 100)),
    CONSTRAINT "chk_worker_training_records_completed" CHECK ("status" NOT IN ('Completed', 'Expired') OR "completed_at" IS NOT NULL),
    CONSTRAINT "chk_worker_training_records_waived" CHECK ("status" <> 'Waived' OR "waived_reason" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_training_records_open" ON "worker_training_records" ("organization_id", "business_unit_id", "worker_id", "course_id")WHERE
    "status" IN ('Assigned', 'InProgress');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_training_records_worker" ON "worker_training_records" ("worker_id", "status", "due_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_training_records_due" ON "worker_training_records" ("due_at")WHERE
    "status" IN ('Assigned', 'InProgress') AND "due_at" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_training_records_expiry" ON "worker_training_records" ("expires_at")WHERE
    "status" = 'Completed' AND "expires_at" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_training_records_document" ON "worker_training_records" ("document_id")WHERE
    "document_id" IS NOT NULL;
