-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261230000100_carrier_intelligence.tx.up.sql

CREATE TABLE IF NOT EXISTS "carrier_intel_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "primary_provider" TEXT,
    "fallback_provider" TEXT,
    "enrollment_policy" TEXT NOT NULL DEFAULT 'Manual',
    "recent_usage_days" INTEGER NOT NULL DEFAULT 90,
    "include_open_tenders" INTEGER NOT NULL DEFAULT 0,
    "auto_enroll_on_create" INTEGER NOT NULL DEFAULT 0,
    "auto_unenroll_on_inactive" INTEGER NOT NULL DEFAULT 0,
    "exclusive_watchlist" INTEGER NOT NULL DEFAULT 0,
    "poll_interval_minutes" INTEGER NOT NULL DEFAULT 120,
    "snapshot_ttl_hours" INTEGER NOT NULL DEFAULT 24,
    "full_profile_ttl_days" INTEGER NOT NULL DEFAULT 30,
    "pretender_refresh_enabled" INTEGER NOT NULL DEFAULT 0,
    "pretender_max_age_hours" INTEGER NOT NULL DEFAULT 24,
    "hard_max_age_hours" INTEGER NOT NULL DEFAULT 168,
    "confirm_blocking_changes" INTEGER NOT NULL DEFAULT 0,
    "outage_policy" TEXT NOT NULL DEFAULT 'FailOpen',
    "auto_disqualify_on_block" INTEGER NOT NULL DEFAULT 0,
    "auto_apply_safety_rating" INTEGER NOT NULL DEFAULT 0,
    "rules" TEXT NOT NULL DEFAULT '{}',
    "auto_sync_fields" TEXT NOT NULL DEFAULT '[]',
    "monthly_spend_cap" REAL,
    "soft_cap_percent" INTEGER NOT NULL DEFAULT 80,
    "daily_full_profile_cap" INTEGER,
    "raw_retention_days" INTEGER NOT NULL DEFAULT 90,
    "snapshot_history_limit" INTEGER NOT NULL DEFAULT 12,
    "self_monitoring_enabled" INTEGER NOT NULL DEFAULT 0,
    "policy_version" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_controls_enrollment_policy" CHECK ("enrollment_policy" IN ('AllActive', 'RecentlyUsed', 'Manual')),
    CONSTRAINT "chk_carrier_intel_controls_outage_policy" CHECK ("outage_policy" IN ('FailOpen', 'FailClosed')),
    CONSTRAINT "chk_carrier_intel_controls_recent_usage_days" CHECK ("recent_usage_days" BETWEEN 7 AND 365),
    CONSTRAINT "chk_carrier_intel_controls_poll_interval" CHECK ("poll_interval_minutes" BETWEEN 60 AND 1440),
    CONSTRAINT "chk_carrier_intel_controls_soft_cap_percent" CHECK ("soft_cap_percent" BETWEEN 1 AND 100),
    CONSTRAINT "chk_carrier_intel_controls_raw_retention_days" CHECK ("raw_retention_days" BETWEEN 7 AND 730),
    CONSTRAINT "chk_carrier_intel_controls_history_limit" CHECK ("snapshot_history_limit" BETWEEN 1 AND 100),
    CONSTRAINT "chk_carrier_intel_controls_spend_cap" CHECK ("monthly_spend_cap" IS NULL OR "monthly_spend_cap" >= 0),
    CONSTRAINT "chk_carrier_intel_controls_fallback" CHECK ("fallback_provider" IS NULL OR "fallback_provider" = 'FMCSAQCMobile')
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_controls_tenant" ON "carrier_intel_controls" ("organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_raw_payloads"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "endpoint" TEXT NOT NULL,
    "dot_number" TEXT,
    "payload" TEXT NOT NULL,
    "byte_size" INTEGER NOT NULL DEFAULT 0,
    "fetched_at" INTEGER NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_raw_payloads" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_raw_payloads_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_raw_payloads_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_raw_payloads_expiry" CHECK ("expires_at" > "fetched_at")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_raw_payloads_expiry" ON "carrier_intel_raw_payloads" ("organization_id", "business_unit_id", "expires_at");

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_snapshots"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "carrier_id" TEXT,
    "dot_number" TEXT NOT NULL,
    "docket_number" TEXT,
    "provider" TEXT NOT NULL,
    "provider_ref" TEXT,
    "depth" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "is_current" INTEGER NOT NULL DEFAULT 0,
    "not_found" INTEGER NOT NULL DEFAULT 0,
    "profile" TEXT NOT NULL,
    "findings" TEXT NOT NULL DEFAULT '[]',
    "blocking_codes" TEXT,
    "advisory_codes" TEXT,
    "risk_level" TEXT NOT NULL DEFAULT 'Unknown',
    "review_state" TEXT NOT NULL DEFAULT 'None',
    "reviewed_by_id" TEXT,
    "reviewed_at" INTEGER,
    "review_note" TEXT,
    "policy_version" INTEGER NOT NULL DEFAULT 0,
    "raw_payload_id" TEXT,
    "content_hash" TEXT NOT NULL,
    "fetched_at" INTEGER NOT NULL,
    "source_as_of" INTEGER,
    "confirmed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_snapshots" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_snapshots_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_reviewed_by" FOREIGN KEY ("reviewed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_carrier_intel_snapshots_subject_type" CHECK ("subject_type" IN ('Carrier', 'Organization', 'Customer', 'Prospect')),
    CONSTRAINT "chk_carrier_intel_snapshots_depth" CHECK ("depth" IN ('FMCSA', 'Lite', 'Full')),
    CONSTRAINT "chk_carrier_intel_snapshots_source" CHECK ("source" IN ('Primary', 'Fallback', 'ChangeFeedPatch')),
    CONSTRAINT "chk_carrier_intel_snapshots_review_state" CHECK ("review_state" IN ('None', 'NeedsReview', 'Reviewed'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_snapshots_current" ON "carrier_intel_snapshots" ("organization_id", "business_unit_id", "subject_type", "subject_id")WHERE
    "is_current";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_subject_history" ON "carrier_intel_snapshots" ("organization_id", "business_unit_id", "subject_type", "subject_id", "fetched_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_carrier" ON "carrier_intel_snapshots" ("organization_id", "business_unit_id", "carrier_id")WHERE
    "is_current" AND "carrier_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_dot" ON "carrier_intel_snapshots" ("organization_id", "dot_number", "fetched_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_needs_review" ON "carrier_intel_snapshots" ("organization_id", "business_unit_id", "fetched_at" DESC)WHERE
    "is_current" AND "review_state" = 'NeedsReview';

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_events"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "carrier_id" TEXT,
    "dot_number" TEXT NOT NULL,
    "subject_name" TEXT,
    "provider" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "field_path" TEXT,
    "rule_code" TEXT,
    "severity" TEXT NOT NULL,
    "action" TEXT,
    "prior_value" TEXT,
    "current_value" TEXT,
    "summary" TEXT NOT NULL,
    "vendor_changed_at" INTEGER,
    "detected_at" INTEGER NOT NULL,
    "fingerprint" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "acknowledged_by_id" TEXT,
    "acknowledged_at" INTEGER,
    "resolved_by_id" TEXT,
    "resolved_at" INTEGER,
    "resolution" TEXT,
    "resolution_note" TEXT,
    "snapshot_id" TEXT,
    "notified_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_acknowledged_by" FOREIGN KEY ("acknowledged_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_carrier_intel_events_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_carrier_intel_events_status" CHECK ("status" IN ('Open', 'Acknowledged', 'Resolved', 'Dismissed')),
    CONSTRAINT "chk_carrier_intel_events_severity" CHECK ("severity" IN ('Critical', 'High', 'Medium', 'Low', 'Info')),
    CONSTRAINT "chk_carrier_intel_events_resolved" CHECK ("status" NOT IN ('Resolved', 'Dismissed') OR ("resolved_at" IS NOT NULL AND "resolution" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_events_fingerprint" ON "carrier_intel_events" ("organization_id", "business_unit_id", "provider", "subject_type", "subject_id", "fingerprint");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_inbox" ON "carrier_intel_events" ("organization_id", "business_unit_id", "status", "detected_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_carrier" ON "carrier_intel_events" ("organization_id", "business_unit_id", "carrier_id", "detected_at" DESC)WHERE
    "carrier_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_subject" ON "carrier_intel_events" ("organization_id", "business_unit_id", "subject_type", "subject_id", "detected_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_monitoring_enrollments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "carrier_id" TEXT,
    "subject_name" TEXT,
    "dot_number" TEXT NOT NULL,
    "docket_number" TEXT,
    "provider" TEXT NOT NULL,
    "provider_ref" TEXT,
    "mode" TEXT NOT NULL,
    "desired_state" TEXT NOT NULL,
    "vendor_state" TEXT NOT NULL DEFAULT 'Unknown',
    "reason" TEXT NOT NULL,
    "owned_by_trenova" INTEGER NOT NULL DEFAULT 0,
    "enrolled_at" INTEGER,
    "unenrolled_at" INTEGER,
    "last_synced_at" INTEGER,
    "last_confirmed_at" INTEGER,
    "last_used_at" INTEGER,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_monitoring_enrollments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_monitoring_enrollments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_monitoring_enrollments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_monitoring_enrollments_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_monitoring_enrollments_mode" CHECK ("mode" IN ('Native', 'SnapshotDiff')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_desired_state" CHECK ("desired_state" IN ('Enrolled', 'NotEnrolled')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_vendor_state" CHECK ("vendor_state" IN ('Unknown', 'PendingAdd', 'Active', 'PendingRemove', 'Removed', 'Failed')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_failure_count" CHECK ("failure_count" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_monitoring_enrollments_subject" ON "carrier_monitoring_enrollments" ("organization_id", "business_unit_id", "provider", "subject_type", "subject_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_pending" ON "carrier_monitoring_enrollments" ("organization_id", "business_unit_id", "provider")WHERE
    "vendor_state" IN ('Unknown', 'PendingAdd', 'PendingRemove') OR ("desired_state" = 'NotEnrolled' AND "vendor_state" = 'Active');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_active" ON "carrier_monitoring_enrollments" ("organization_id", "business_unit_id", "provider", "mode", "last_confirmed_at")WHERE
    "desired_state" = 'Enrolled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_carrier" ON "carrier_monitoring_enrollments" ("organization_id", "business_unit_id", "carrier_id")WHERE
    "carrier_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_feed_states"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "feed_type" TEXT NOT NULL,
    "cursor" TEXT NOT NULL DEFAULT '{}',
    "last_polled_at" INTEGER,
    "last_success_at" INTEGER,
    "next_poll_after" INTEGER,
    "paused_reason" TEXT,
    "paused_at" INTEGER,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    CONSTRAINT "pk_carrier_intel_feed_states" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "feed_type"),
    CONSTRAINT "fk_carrier_intel_feed_states_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_feed_states_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_feed_states_failure_count" CHECK ("failure_count" >= 0),
    CONSTRAINT "chk_carrier_intel_feed_states_paused_reason" CHECK ("paused_reason" IS NULL OR "paused_reason" IN ('PaymentRequired', 'SpendCap', 'Unauthorized', 'Disabled'))
);

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_equipment_verifications"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_assignment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT,
    "carrier_id" TEXT NOT NULL,
    "expected_dot_number" TEXT,
    "unit_type" TEXT NOT NULL,
    "vin" TEXT,
    "plate_number" TEXT,
    "plate_state" TEXT,
    "unit_number" TEXT,
    "result" TEXT NOT NULL,
    "matched_dot_numbers" TEXT,
    "matched_legal_name" TEXT,
    "detail" TEXT,
    "mismatch_reason" TEXT,
    "provider" TEXT,
    "verified_by_id" TEXT NOT NULL,
    "verified_at" INTEGER NOT NULL,
    "override_by_id" TEXT,
    "override_reason" TEXT,
    "overridden_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_equipment_verifications" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_equipment_verifications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_verified_by" FOREIGN KEY ("verified_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_equipment_verifications_override_by" FOREIGN KEY ("override_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_equipment_verifications_identifier" CHECK ("vin" IS NOT NULL OR ("plate_number" IS NOT NULL AND "plate_state" IS NOT NULL) OR "unit_number" IS NOT NULL),
    CONSTRAINT "chk_carrier_equipment_verifications_result" CHECK ("result" IN ('Match', 'Mismatch', 'NotFound', 'Unverifiable', 'ProviderError')),
    CONSTRAINT "chk_carrier_equipment_verifications_unit_type" CHECK ("unit_type" IN ('Tractor', 'Trailer', 'Straight')),
    CONSTRAINT "chk_carrier_equipment_verifications_override" CHECK ("overridden_at" IS NULL OR ("override_by_id" IS NOT NULL AND "override_reason" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_equipment_verifications_assignment" ON "carrier_equipment_verifications" ("organization_id", "business_unit_id", "carrier_assignment_id", "verified_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_equipment_verifications_carrier" ON "carrier_equipment_verifications" ("organization_id", "business_unit_id", "carrier_id", "verified_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_usage_ledger"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "endpoint" TEXT NOT NULL,
    "purpose" TEXT NOT NULL,
    "dot_number" TEXT,
    "carrier_id" TEXT,
    "billable" INTEGER NOT NULL DEFAULT 0,
    "billable_units" INTEGER NOT NULL DEFAULT 0,
    "estimated_cost" REAL NOT NULL DEFAULT 0,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "dedupe_key" TEXT,
    "status_code" INTEGER NOT NULL DEFAULT 0,
    "outcome" TEXT NOT NULL,
    "latency_ms" INTEGER NOT NULL DEFAULT 0,
    "initiated_by_type" TEXT NOT NULL DEFAULT 'System',
    "initiated_by_id" TEXT,
    "workflow_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_usage_ledger" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_usage_ledger_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_usage_ledger_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_usage_ledger_cost" CHECK ("estimated_cost" >= 0 AND "billable_units" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_usage_ledger_tenant_created" ON "carrier_intel_usage_ledger" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_usage_ledger_dedupe" ON "carrier_intel_usage_ledger" ("organization_id", "business_unit_id", "provider", "dedupe_key")WHERE
    "billable" AND "dedupe_key" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_usage_daily"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "endpoint" TEXT NOT NULL,
    "day" INTEGER NOT NULL,
    "calls" INTEGER NOT NULL DEFAULT 0,
    "billable_units" INTEGER NOT NULL DEFAULT 0,
    "estimated_cost" REAL NOT NULL DEFAULT 0,
    CONSTRAINT "pk_carrier_intel_usage_daily" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "endpoint", "day"),
    CONSTRAINT "fk_carrier_intel_usage_daily_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_usage_daily_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_intel_overrides"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "rule_code" TEXT NOT NULL,
    "reason" TEXT NOT NULL,
    "granted_by_id" TEXT NOT NULL,
    "granted_at" INTEGER NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "revoked_by_id" TEXT,
    "revoked_at" INTEGER,
    "revoke_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_intel_overrides" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_granted_by" FOREIGN KEY ("granted_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_intel_overrides_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_intel_overrides_window" CHECK ("expires_at" > "granted_at" AND "expires_at" - "granted_at" <= 7776000),
    CONSTRAINT "chk_carrier_intel_overrides_revoked" CHECK ("revoked_at" IS NULL OR ("revoked_by_id" IS NOT NULL AND "revoke_reason" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_intel_overrides_active" ON "carrier_intel_overrides" ("organization_id", "business_unit_id", "carrier_id", "expires_at")WHERE
    "revoked_at" IS NULL;
