-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261013000000_worker_safety_performance.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_safety_events"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'Minor',
    "status" TEXT NOT NULL DEFAULT 'Open',
    "occurred_at" INTEGER NOT NULL,
    "location" TEXT,
    "description" TEXT NOT NULL,
    "preventable" INTEGER NOT NULL DEFAULT 0,
    "points" INTEGER NOT NULL DEFAULT 0,
    "points_expire_at" INTEGER,
    "reference_number" TEXT,
    "shipment_id" TEXT,
    "inspection_level" INTEGER,
    "inspection_result" TEXT,
    "out_of_service" INTEGER NOT NULL DEFAULT 0,
    "fine_amount" REAL,
    "cost_amount" REAL,
    "document_id" TEXT,
    "recorded_by_id" TEXT,
    "closed_by_id" TEXT,
    "closed_at" INTEGER,
    "resolution" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_safety_events" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_safety_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_events_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_events_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_safety_events_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_safety_events_points" CHECK ("points" >= 0),
    CONSTRAINT "chk_worker_safety_events_level" CHECK ("inspection_level" IS NULL OR ("inspection_level" BETWEEN 1 AND 6)),
    CONSTRAINT "chk_worker_safety_events_closed" CHECK (("status" = 'Closed') = ("closed_at" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_worker" ON "worker_safety_events" ("worker_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_points" ON "worker_safety_events" ("points_expire_at")WHERE
    "points" > 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_shipment" ON "worker_safety_events" ("shipment_id")WHERE
    "shipment_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "worker_disciplinary_actions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "level" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "reason" TEXT NOT NULL,
    "details" TEXT,
    "occurred_at" INTEGER,
    "issued_at" INTEGER NOT NULL,
    "expires_at" INTEGER,
    "suspension_days" INTEGER,
    "safety_event_id" TEXT,
    "document_id" TEXT,
    "issued_by_id" TEXT,
    "acknowledged_at" INTEGER,
    "worker_comment" TEXT,
    "rescinded_at" INTEGER,
    "rescinded_by_id" TEXT,
    "rescind_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_disciplinary_actions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_disciplinary_actions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_disciplinary_actions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_disciplinary_actions_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_disciplinary_actions_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_disciplinary_actions_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_disciplinary_actions_suspension" CHECK ("suspension_days" IS NULL OR "suspension_days" > 0),
    CONSTRAINT "chk_worker_disciplinary_actions_rescinded" CHECK (("status" = 'Rescinded') = ("rescinded_at" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_disciplinary_actions_worker" ON "worker_disciplinary_actions" ("worker_id", "status", "issued_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_disciplinary_actions_expiry" ON "worker_disciplinary_actions" ("expires_at")WHERE
    "status" = 'Active' AND "expires_at" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "worker_recognitions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL DEFAULT 'Other',
    "title" TEXT NOT NULL,
    "message" TEXT,
    "occurred_at" INTEGER NOT NULL,
    "awarded_by_id" TEXT,
    "visible_to_worker" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_recognitions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_recognitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_recognitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_recognitions_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_recognitions_worker" ON "worker_recognitions" ("worker_id", "occurred_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "performance_review_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "cadence_months" INTEGER,
    "items" TEXT NOT NULL DEFAULT '[]',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_performance_review_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_performance_review_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_performance_review_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_performance_review_templates_cadence" CHECK ("cadence_months" IS NULL OR "cadence_months" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_review_templates_code" ON "performance_review_templates" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_review_templates_default" ON "performance_review_templates" ("organization_id", "business_unit_id")WHERE
    "is_default" = TRUE;

--bun:split

CREATE TABLE IF NOT EXISTS "performance_reviews"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "reviewer_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "title" TEXT NOT NULL,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "ratings" TEXT NOT NULL DEFAULT '[]',
    "overall_score" REAL,
    "summary" TEXT,
    "strengths" TEXT,
    "improvements" TEXT,
    "goals" TEXT NOT NULL DEFAULT '[]',
    "submitted_at" INTEGER,
    "acknowledged_at" INTEGER,
    "worker_comment" TEXT,
    "closed_at" INTEGER,
    "closed_by_id" TEXT,
    "next_review_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_performance_reviews" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_performance_reviews_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_performance_reviews_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_performance_reviews_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_performance_reviews_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "performance_review_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_performance_reviews_period" CHECK ("period_end" >= "period_start"),
    CONSTRAINT "chk_performance_reviews_score" CHECK ("overall_score" IS NULL OR ("overall_score" >= 0 AND "overall_score" <= 5)),
    CONSTRAINT "chk_performance_reviews_submitted" CHECK ("status" = 'Draft' OR "submitted_at" IS NOT NULL),
    CONSTRAINT "chk_performance_reviews_closed" CHECK (("status" = 'Closed') = ("closed_at" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_reviews_open" ON "performance_reviews" ("organization_id", "business_unit_id", "worker_id", "template_id")WHERE
    "status" IN ('Draft', 'Submitted');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_performance_reviews_worker" ON "performance_reviews" ("worker_id", "status", "period_end" DESC);
