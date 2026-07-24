--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
CREATE TYPE "agent_type_enum" AS ENUM(
    'BillingException'
);

CREATE TYPE "agent_subject_type_enum" AS ENUM(
    'BillingQueueItem'
);

CREATE TYPE "agent_run_status_enum" AS ENUM(
    'Pending',
    'GatheringContext',
    'Diagnosing',
    'AwaitingDecision',
    'Completed',
    'ShadowCompleted',
    'Failed'
);

CREATE TYPE "agent_proposal_status_enum" AS ENUM(
    'Pending',
    'Accepted',
    'Modified',
    'Rejected',
    'Expired',
    'Superseded'
);

CREATE TYPE "agent_autonomy_tier_enum" AS ENUM(
    'Propose',
    'ActWithApproval',
    'AutoExecute'
);

CREATE TYPE "agent_exception_category_enum" AS ENUM(
    'MissingDocumentation',
    'IncorrectRates',
    'WeightDiscrepancy',
    'AccessorialDispute',
    'DuplicateCharge',
    'MissingReferenceNumber',
    'CustomerInformationError',
    'ServiceFailure',
    'RateNotOnFile',
    'MissingBOL',
    'RateMissingBasis',
    'RateVarianceRequiresAction',
    'UnresolvedServiceFailures',
    'MissingRequiredDocument',
    'ConfidenceBelowThreshold',
    'UnableToDiagnose',
    'Other'
);

CREATE TYPE "agent_severity_enum" AS ENUM(
    'Low',
    'Medium',
    'High',
    'Critical'
);

CREATE TYPE "agent_resolution_state_enum" AS ENUM(
    'Open',
    'InReview',
    'Resolved',
    'Dismissed'
);

CREATE TYPE "agent_decision_type_enum" AS ENUM(
    'Accepted',
    'Modified',
    'Rejected'
);

--bun:split
CREATE TABLE IF NOT EXISTS "agent_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shadow_mode" boolean NOT NULL DEFAULT TRUE,
    "billing_agent_enabled" boolean NOT NULL DEFAULT FALSE,
    "decision_timeout_seconds" integer NOT NULL DEFAULT 86400,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_controls_decision_timeout" CHECK ("decision_timeout_seconds" >= 60),
    CONSTRAINT "fk_agent_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_controls_tenant" ON "agent_controls"("organization_id", "business_unit_id");

--bun:split
CREATE TABLE IF NOT EXISTS "agent_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_type" agent_type_enum NOT NULL,
    "subject_type" agent_subject_type_enum NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "status" agent_run_status_enum NOT NULL DEFAULT 'Pending',
    "workflow_id" varchar(255),
    "model_identifier" varchar(255),
    "prompt_version" varchar(100) NOT NULL,
    "input_context_hash" varchar(64) NOT NULL,
    "started_at" bigint,
    "completed_at" bigint,
    "error_message" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_runs_subject" ON "agent_runs"("organization_id", "subject_type", "subject_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_runs_workflow" ON "agent_runs"("workflow_id");

--bun:split
CREATE TABLE IF NOT EXISTS "agent_proposals"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "tool_name" varchar(100) NOT NULL,
    "tool_params" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "confidence" numeric(5, 4) NOT NULL DEFAULT 0,
    "rationale" text NOT NULL,
    "evidence" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "autonomy_tier" agent_autonomy_tier_enum NOT NULL DEFAULT 'Propose',
    "status" agent_proposal_status_enum NOT NULL DEFAULT 'Pending',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_proposals" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_proposals_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_run_status" ON "agent_proposals"("organization_id", "run_id", "status");

--bun:split
CREATE TABLE IF NOT EXISTS "agent_exceptions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "category" agent_exception_category_enum NOT NULL,
    "severity" agent_severity_enum NOT NULL DEFAULT 'Medium',
    "subject_type" agent_subject_type_enum NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "attempt_summary" text NOT NULL,
    "evidence" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "blast_radius" integer NOT NULL DEFAULT 0,
    "resolution_state" agent_resolution_state_enum NOT NULL DEFAULT 'Open',
    "resolution_notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_exceptions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_exceptions_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_exceptions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_exceptions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_exceptions_queue" ON "agent_exceptions"("organization_id", "resolution_state", "severity");

--bun:split
CREATE TABLE IF NOT EXISTS "agent_decisions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "proposal_id" varchar(100),
    "exception_id" varchar(100),
    "decided_by_user_id" varchar(100) NOT NULL,
    "decision" agent_decision_type_enum NOT NULL,
    "modifications" jsonb,
    "reason_code" varchar(100) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_decisions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_decisions_subject" CHECK (("proposal_id" IS NOT NULL)::int + ("exception_id" IS NOT NULL)::int = 1),
    CONSTRAINT "fk_agent_decisions_proposal" FOREIGN KEY ("proposal_id", "business_unit_id", "organization_id") REFERENCES "agent_proposals"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_exception" FOREIGN KEY ("exception_id", "business_unit_id", "organization_id") REFERENCES "agent_exceptions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_user" FOREIGN KEY ("decided_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_decisions_proposal" ON "agent_decisions"("organization_id", "proposal_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_decisions_exception" ON "agent_decisions"("organization_id", "exception_id");
