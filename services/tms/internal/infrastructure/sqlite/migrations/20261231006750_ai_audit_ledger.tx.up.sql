-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006750_ai_audit_ledger.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_audit_events"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "seq" INTEGER NOT NULL,
    "source_key" TEXT NOT NULL,
    "occurred_at" INTEGER NOT NULL,
    "recorded_at" INTEGER NOT NULL,
    "kind" TEXT NOT NULL,
    "outcome" TEXT NOT NULL,
    "principal_type" TEXT NOT NULL,
    "principal_id" TEXT,
    "on_behalf_of_user_id" TEXT,
    "on_behalf_of_user_name" TEXT,
    "decided_by_user_id" TEXT,
    "decided_by_user_name" TEXT,
    "agent_definition_id" TEXT,
    "agent_definition_version" INTEGER,
    "agent_name" TEXT,
    "owner_kind" TEXT,
    "owner_id" TEXT,
    "run_id" TEXT,
    "turn_id" TEXT,
    "thread_id" TEXT,
    "proposal_id" TEXT,
    "plan_id" TEXT,
    "decision_id" TEXT,
    "step_key" TEXT,
    "call_id" TEXT,
    "delegate_call_id" TEXT,
    "parent_owner_id" TEXT,
    "trace_id" TEXT,
    "span_id" TEXT,
    "provider_id" TEXT,
    "provider_kind" TEXT,
    "model" TEXT,
    "attempt" INTEGER,
    "failover" INTEGER NOT NULL DEFAULT 0,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "reasoning_tokens" INTEGER NOT NULL DEFAULT 0,
    "cache_read_tokens" INTEGER NOT NULL DEFAULT 0,
    "cache_write_tokens" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" REAL,
    "latency_ms" INTEGER,
    "tool_name" TEXT,
    "tool_effect" TEXT,
    "egress_class" TEXT,
    "tier" TEXT,
    "tier_source" TEXT,
    "held_by" TEXT NOT NULL DEFAULT '[]',
    "reason" TEXT,
    "arguments" TEXT,
    "argument_sensitivity" TEXT,
    "redacted_paths" TEXT NOT NULL DEFAULT '[]',
    "arguments_truncated" INTEGER NOT NULL DEFAULT 0,
    "result_summary" TEXT,
    "entity_type" TEXT,
    "entity_id" TEXT,
    "version_before" INTEGER,
    "version_after" INTEGER,
    "window_start" INTEGER,
    "window_end" INTEGER,
    "tainted" INTEGER NOT NULL DEFAULT 0,
    "taint" TEXT,
    "external_content" INTEGER NOT NULL DEFAULT 0,
    "simulated" INTEGER NOT NULL DEFAULT 0,
    "purpose" TEXT NOT NULL DEFAULT 'Live',
    "reconstructed" INTEGER NOT NULL DEFAULT 0,
    "prev_hash" TEXT NOT NULL,
    "hash" TEXT NOT NULL,
    "hash_key_id" TEXT,
    "hash_version" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_audit_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_ai_audit_events_kind" CHECK ("kind" IN ('RunStarted', 'RunEnded', 'ModelCall', 'ToolCall', 'ToolRefused', 'ProposalFiled', 'ProposalDecided', 'ProposalExecuted', 'ProposalExecutionFailed', 'ProposalSimulated', 'ProposalExpired', 'DelegationStarted', 'DelegationEnded')),
    CONSTRAINT "ck_ai_audit_events_outcome" CHECK ("outcome" IN ('Started', 'Completed', 'Failed', 'Refused', 'Stopped', 'Succeeded', 'Ran', 'Proposed', 'Simulated', 'Denied', 'Unknown', 'Filed', 'Accepted', 'Modified', 'Rejected', 'Expired', 'Exhausted', 'Declined')),
    CONSTRAINT "ck_ai_audit_events_principal_type" CHECK ("principal_type" IN ('User', 'Agent', 'System')),
    CONSTRAINT "ck_ai_audit_events_purpose" CHECK ("purpose" IN ('Live', 'Evaluation')),
    CONSTRAINT "ck_ai_audit_events_owner_kind" CHECK ("owner_kind" IS NULL OR "owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_ai_audit_events_hash_version" CHECK ("hash_version" IN (1, 2)),
    CONSTRAINT "ck_ai_audit_events_seq" CHECK ("seq" >= 1),
    CONSTRAINT "ck_ai_audit_events_signed_key" CHECK (("hash_version" = 1) = ("hash_key_id" IS NOT NULL)),
    CONSTRAINT "fk_ai_audit_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_source_key" ON "ai_audit_events" ("organization_id", "business_unit_id", "source_key");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_seq" ON "ai_audit_events" ("organization_id", "business_unit_id", "seq");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_occurred" ON "ai_audit_events" ("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_agent" ON "ai_audit_events" ("organization_id", "business_unit_id", "agent_definition_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_tool" ON "ai_audit_events" ("organization_id", "business_unit_id", "tool_name", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_entity" ON "ai_audit_events" ("organization_id", "business_unit_id", "entity_type", "entity_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_on_behalf_of" ON "ai_audit_events" ("organization_id", "business_unit_id", "on_behalf_of_user_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_decided_by" ON "ai_audit_events" ("organization_id", "business_unit_id", "decided_by_user_id")WHERE "decided_by_user_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_owner" ON "ai_audit_events" ("organization_id", "business_unit_id", "owner_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_trace" ON "ai_audit_events" ("organization_id", "business_unit_id", "trace_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_proposal" ON "ai_audit_events" ("organization_id", "business_unit_id", "proposal_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_created" ON "ai_audit_events" ("created_at");

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_chain_heads"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "last_seq" INTEGER NOT NULL DEFAULT 0,
    "last_hash" TEXT NOT NULL DEFAULT '',
    "hash_key_id" TEXT,
    "last_verified_seq" INTEGER NOT NULL DEFAULT 0,
    "last_verified_at" INTEGER,
    "last_verification_status" TEXT,
    "last_verification_failed_seq" INTEGER,
    "last_verification_detail" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_audit_chain_heads" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Mismatch', 'KeyMissing')),
    CONSTRAINT "ck_ai_audit_chain_heads_last_seq" CHECK ("last_seq" >= 0),
    CONSTRAINT "fk_ai_audit_chain_heads_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_chain_heads_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_seals"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "to_seq" INTEGER NOT NULL,
    "from_seq" INTEGER NOT NULL,
    "head_hash" TEXT NOT NULL,
    "hash_key_id" TEXT,
    "row_count" INTEGER NOT NULL,
    "sealed_at" INTEGER NOT NULL,
    CONSTRAINT "pk_ai_audit_seals" PRIMARY KEY ("organization_id", "business_unit_id", "to_seq"),
    CONSTRAINT "ck_ai_audit_seals_range" CHECK ("from_seq" >= 1 AND "to_seq" >= "from_seq" AND "row_count" = "to_seq" - "from_seq" + 1),
    CONSTRAINT "fk_ai_audit_seals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_seals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_seals_sealed" ON "ai_audit_seals" ("organization_id", "business_unit_id", "sealed_at");

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_exports"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "requested_by_user_id" TEXT NOT NULL,
    "format" TEXT NOT NULL,
    "filters" TEXT NOT NULL DEFAULT '{}',
    "range_from" INTEGER NOT NULL,
    "range_to" INTEGER NOT NULL,
    "snapshot_seq" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "row_count" INTEGER NOT NULL DEFAULT 0,
    "byte_size" INTEGER NOT NULL DEFAULT 0,
    "sha256" TEXT,
    "artifact_key" TEXT,
    "artifact_expires_at" INTEGER,
    "chain_key_id" TEXT,
    "chain_first_seq" INTEGER,
    "chain_last_seq" INTEGER,
    "chain_complete" INTEGER NOT NULL DEFAULT 0,
    "workflow_id" TEXT,
    "error_message" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_audit_exports" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_ai_audit_exports_format" CHECK ("format" IN ('CSV', 'JSON')),
    CONSTRAINT "ck_ai_audit_exports_status" CHECK ("status" IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Expired')),
    CONSTRAINT "ck_ai_audit_exports_range" CHECK ("range_to" >= "range_from"),
    CONSTRAINT "fk_ai_audit_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_exports_listing" ON "ai_audit_exports" ("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_exports_expiry" ON "ai_audit_exports" ("artifact_expires_at")WHERE "artifact_key" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_projector_state"(
    "source" TEXT NOT NULL,
    "watermark_ts" INTEGER NOT NULL DEFAULT 0,
    "watermark_id" TEXT NOT NULL DEFAULT '',
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_audit_projector_state" PRIMARY KEY ("source"),
    CONSTRAINT "ck_ai_audit_projector_state_source" CHECK ("source" IN ('agent_runs', 'assistant_turns', 'ai_usage_records', 'agent_run_steps', 'agent_run_events', 'agent_proposals', 'agent_decisions'))
);
