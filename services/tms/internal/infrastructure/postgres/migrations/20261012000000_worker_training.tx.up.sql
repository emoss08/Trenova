--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "training_category_enum" AS ENUM(
    'Safety',
    'Compliance',
    'Equipment',
    'Orientation',
    'HazardousMaterials',
    'Other'
);

CREATE TYPE "training_delivery_enum" AS ENUM(
    'Online',
    'Classroom',
    'OnTheJob',
    'Document'
);

CREATE TYPE "worker_training_status_enum" AS ENUM(
    'Assigned',
    'InProgress',
    'Completed',
    'Failed',
    'Expired',
    'Waived',
    'Cancelled'
);

--bun:split
CREATE TABLE IF NOT EXISTS "training_courses"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "category" training_category_enum NOT NULL DEFAULT 'Other',
    "status" status_enum NOT NULL DEFAULT 'Active',
    "delivery" training_delivery_enum NOT NULL DEFAULT 'Online',
    "content_url" varchar(500),
    "duration_minutes" integer NOT NULL DEFAULT 0,
    "passing_score" numeric(5, 2),
    "validity_months" integer,
    "renewal_window_days" integer NOT NULL DEFAULT 30,
    "is_required" boolean NOT NULL DEFAULT FALSE,
    "required_for_driver_types" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "due_days_after_assignment" integer NOT NULL DEFAULT 30,
    "requires_acknowledgement" boolean NOT NULL DEFAULT TRUE,
    "sort_order" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
ALTER TABLE "training_courses"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_training_courses_code" ON "training_courses"("organization_id", "business_unit_id", LOWER("code"));
--bun:split
CREATE INDEX IF NOT EXISTS "idx_training_courses_status" ON "training_courses"("organization_id", "business_unit_id", "status", "sort_order");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_training_courses_search" ON "training_courses" USING GIN(search_vector);
--bun:split
COMMENT ON TABLE training_courses IS 'Catalog of training a worker can be assigned. Required courses form the per-driver-type matrix; validity_months turns a completion into a recurring certification.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_training_records"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "course_id" varchar(100) NOT NULL,
    "status" worker_training_status_enum NOT NULL DEFAULT 'Assigned',
    "assigned_at" bigint NOT NULL,
    "due_at" bigint,
    "started_at" bigint,
    "completed_at" bigint,
    "expires_at" bigint,
    "score" numeric(5, 2),
    "passed" boolean,
    "acknowledged_at" bigint,
    "document_id" varchar(100),
    "assigned_by_id" varchar(100),
    "recorded_by_id" varchar(100),
    "notes" text,
    "waived_reason" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_training_records_open" ON "worker_training_records"("organization_id", "business_unit_id", "worker_id", "course_id")
WHERE
    "status" IN ('Assigned', 'InProgress');
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_training_records_worker" ON "worker_training_records"("worker_id", "status", "due_at");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_training_records_due" ON "worker_training_records"("due_at")
WHERE
    "status" IN ('Assigned', 'InProgress') AND "due_at" IS NOT NULL;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_training_records_expiry" ON "worker_training_records"("expires_at")
WHERE
    "status" = 'Completed' AND "expires_at" IS NOT NULL;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_training_records_document" ON "worker_training_records"("document_id")
WHERE
    "document_id" IS NOT NULL;
--bun:split
COMMENT ON TABLE worker_training_records IS 'One row per assignment of a course to a worker. At most one open (Assigned/InProgress) row per worker and course; completions keep their row so history and expiry are tracked.';
