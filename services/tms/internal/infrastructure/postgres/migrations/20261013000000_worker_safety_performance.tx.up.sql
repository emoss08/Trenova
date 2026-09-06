--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "safety_event_kind_enum" AS ENUM(
    'Accident',
    'Incident',
    'NearMiss',
    'Citation',
    'Inspection'
);

CREATE TYPE "safety_severity_enum" AS ENUM(
    'Minor',
    'Moderate',
    'Major',
    'Critical'
);

CREATE TYPE "safety_event_status_enum" AS ENUM(
    'Open',
    'UnderReview',
    'Closed'
);

CREATE TYPE "inspection_result_enum" AS ENUM(
    'Pass',
    'Fail',
    'OutOfService'
);

CREATE TYPE "disciplinary_level_enum" AS ENUM(
    'Coaching',
    'VerbalWarning',
    'WrittenWarning',
    'FinalWarning',
    'Suspension',
    'Termination'
);

CREATE TYPE "disciplinary_status_enum" AS ENUM(
    'Active',
    'Expired',
    'Rescinded'
);

CREATE TYPE "recognition_kind_enum" AS ENUM(
    'SafetyMilestone',
    'CustomerPraise',
    'Performance',
    'Tenure',
    'TeamPlayer',
    'Other'
);

CREATE TYPE "performance_review_status_enum" AS ENUM(
    'Draft',
    'Submitted',
    'Acknowledged',
    'Closed'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_safety_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "kind" safety_event_kind_enum NOT NULL,
    "severity" safety_severity_enum NOT NULL DEFAULT 'Minor',
    "status" safety_event_status_enum NOT NULL DEFAULT 'Open',
    "occurred_at" bigint NOT NULL,
    "location" varchar(255),
    "description" text NOT NULL,
    "preventable" boolean NOT NULL DEFAULT FALSE,
    "points" integer NOT NULL DEFAULT 0,
    "points_expire_at" bigint,
    "reference_number" varchar(100),
    "shipment_id" varchar(100),
    "inspection_level" smallint,
    "inspection_result" inspection_result_enum,
    "out_of_service" boolean NOT NULL DEFAULT FALSE,
    "fine_amount" numeric(12, 2),
    "cost_amount" numeric(12, 2),
    "document_id" varchar(100),
    "recorded_by_id" varchar(100),
    "closed_by_id" varchar(100),
    "closed_at" bigint,
    "resolution" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_worker" ON "worker_safety_events"("worker_id", "occurred_at" DESC);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_points" ON "worker_safety_events"("points_expire_at")
WHERE
    "points" > 0;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_shipment" ON "worker_safety_events"("shipment_id")
WHERE
    "shipment_id" IS NOT NULL;
--bun:split
COMMENT ON TABLE worker_safety_events IS 'Accidents, incidents, near misses, citations and roadside inspections per worker. Points feed the safety scorecard until points_expire_at.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_disciplinary_actions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "level" disciplinary_level_enum NOT NULL,
    "status" disciplinary_status_enum NOT NULL DEFAULT 'Active',
    "reason" text NOT NULL,
    "details" text,
    "occurred_at" bigint,
    "issued_at" bigint NOT NULL,
    "expires_at" bigint,
    "suspension_days" integer,
    "safety_event_id" varchar(100),
    "document_id" varchar(100),
    "issued_by_id" varchar(100),
    "acknowledged_at" bigint,
    "worker_comment" text,
    "rescinded_at" bigint,
    "rescinded_by_id" varchar(100),
    "rescind_reason" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_worker_disciplinary_actions_worker" ON "worker_disciplinary_actions"("worker_id", "status", "issued_at" DESC);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_disciplinary_actions_expiry" ON "worker_disciplinary_actions"("expires_at")
WHERE
    "status" = 'Active' AND "expires_at" IS NOT NULL;
--bun:split
COMMENT ON TABLE worker_disciplinary_actions IS 'Progressive discipline per worker. Active actions inside the lookback window decide the next rung of the ladder; they expire on expires_at unless rescinded.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_recognitions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "kind" recognition_kind_enum NOT NULL DEFAULT 'Other',
    "title" varchar(120) NOT NULL,
    "message" text,
    "occurred_at" bigint NOT NULL,
    "awarded_by_id" varchar(100),
    "visible_to_worker" boolean NOT NULL DEFAULT TRUE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_recognitions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_recognitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_recognitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_recognitions_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_recognitions_worker" ON "worker_recognitions"("worker_id", "occurred_at" DESC);
--bun:split
COMMENT ON TABLE worker_recognitions IS 'Praise and milestones recorded against a worker; visible_to_worker rows show up as kudos in Dash.';
--bun:split
CREATE TABLE IF NOT EXISTS "performance_review_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "cadence_months" integer,
    "items" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_performance_review_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_performance_review_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_performance_review_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_performance_review_templates_cadence" CHECK ("cadence_months" IS NULL OR "cadence_months" > 0)
);
--bun:split
ALTER TABLE "performance_review_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_review_templates_code" ON "performance_review_templates"("organization_id", "business_unit_id", LOWER("code"));
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_review_templates_default" ON "performance_review_templates"("organization_id", "business_unit_id")
WHERE
    "is_default" = TRUE;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_performance_review_templates_search" ON "performance_review_templates" USING GIN(search_vector);
--bun:split
COMMENT ON TABLE performance_review_templates IS 'Rating items (with weights) a review is scored against. Items are copied onto each review so later edits do not rewrite history.';
--bun:split
CREATE TABLE IF NOT EXISTS "performance_reviews"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "reviewer_id" varchar(100),
    "status" performance_review_status_enum NOT NULL DEFAULT 'Draft',
    "title" varchar(120) NOT NULL,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "ratings" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "overall_score" numeric(4, 2),
    "summary" text,
    "strengths" text,
    "improvements" text,
    "goals" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "submitted_at" bigint,
    "acknowledged_at" bigint,
    "worker_comment" text,
    "closed_at" bigint,
    "closed_by_id" varchar(100),
    "next_review_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_performance_reviews_open" ON "performance_reviews"("organization_id", "business_unit_id", "worker_id", "template_id")
WHERE
    "status" IN ('Draft', 'Submitted');
--bun:split
CREATE INDEX IF NOT EXISTS "idx_performance_reviews_worker" ON "performance_reviews"("worker_id", "status", "period_end" DESC);
--bun:split
COMMENT ON TABLE performance_reviews IS 'One review per worker and period. Draft until the reviewer submits; the worker signs off in Dash; closing sets the next review date from the template cadence.';
