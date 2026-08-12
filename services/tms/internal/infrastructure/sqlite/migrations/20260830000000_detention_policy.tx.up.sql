-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260830000000_detention_policy.tx.up.sql

CREATE TABLE IF NOT EXISTS "detention_policies"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "is_org_default" INTEGER NOT NULL DEFAULT 0,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "specificity_score" INTEGER NOT NULL DEFAULT 0,
    "customer_id" TEXT,
    "location_id" TEXT,
    "shipment_type_ids" TEXT,
    "service_type_ids" TEXT,
    "commodity_ids" TEXT,
    "stop_types" TEXT,
    "appointment_stops_only" INTEGER NOT NULL DEFAULT 0,
    "effective_start_date" INTEGER,
    "effective_end_date" INTEGER,
    "clock_start_basis" TEXT NOT NULL DEFAULT 'LaterOfArrivalOrAppointment',
    "late_arrival_rule" TEXT NOT NULL DEFAULT 'NoEffect',
    "late_arrival_grace_minutes" INTEGER NOT NULL DEFAULT 0,
    "billing_free_minutes" INTEGER NOT NULL DEFAULT 120,
    "pickup_free_minutes" INTEGER,
    "delivery_free_minutes" INTEGER,
    "pay_free_minutes" INTEGER,
    "minimum_billable_minutes" INTEGER NOT NULL DEFAULT 0,
    "billing_increment_minutes" INTEGER NOT NULL DEFAULT 15,
    "rounding_mode" TEXT NOT NULL DEFAULT 'Up',
    "rate_source" TEXT NOT NULL DEFAULT 'Accessorial',
    "accessorial_charge_id" TEXT NOT NULL,
    "max_billable_minutes_per_stop" INTEGER,
    "max_charge_per_stop" NUMERIC,
    "max_charge_per_day" NUMERIC,
    "max_charge_per_shipment" NUMERIC,
    "day_boundary_mode" TEXT NOT NULL DEFAULT 'PerStop',
    "convert_to_layover_at_minutes" INTEGER,
    "layover_accessorial_charge_id" TEXT,
    "notification_requirement" TEXT NOT NULL DEFAULT 'None',
    "notification_lead_minutes" INTEGER NOT NULL DEFAULT 30,
    "notification_deadline_minutes" INTEGER NOT NULL DEFAULT 0,
    "unnotified_behavior" TEXT NOT NULL DEFAULT 'Bill',
    "auto_send_notice" INTEGER NOT NULL DEFAULT 0,
    "send_departure_summary" INTEGER NOT NULL DEFAULT 0,
    "require_approval_over_amount" NUMERIC,
    "auto_approve_under_amount" NUMERIC,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "comments" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_detention_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policies_accessorial" FOREIGN KEY ("accessorial_charge_id", "organization_id", "business_unit_id") REFERENCES "accessorial_charges"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_detention_policies_layover_accessorial" FOREIGN KEY ("layover_accessorial_charge_id", "organization_id", "business_unit_id") REFERENCES "accessorial_charges"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_detention_policies_free_minutes" CHECK ("billing_free_minutes" BETWEEN 0 AND 10080),
    CONSTRAINT "chk_detention_policies_pickup_free" CHECK ("pickup_free_minutes" IS NULL OR "pickup_free_minutes" BETWEEN 0 AND 10080),
    CONSTRAINT "chk_detention_policies_delivery_free" CHECK ("delivery_free_minutes" IS NULL OR "delivery_free_minutes" BETWEEN 0 AND 10080),
    CONSTRAINT "chk_detention_policies_pay_free" CHECK ("pay_free_minutes" IS NULL OR "pay_free_minutes" BETWEEN 0 AND 10080),
    CONSTRAINT "chk_detention_policies_increment" CHECK (("rounding_mode" = 'Exact' AND "billing_increment_minutes" >= 0) OR ("rounding_mode" <> 'Exact' AND "billing_increment_minutes" BETWEEN 1 AND 1440)),
    CONSTRAINT "chk_detention_policies_min_billable" CHECK ("minimum_billable_minutes" >= 0),
    CONSTRAINT "chk_detention_policies_grace" CHECK ("late_arrival_grace_minutes" >= 0),
    CONSTRAINT "chk_detention_policies_effective_window" CHECK ("effective_start_date" IS NULL OR "effective_end_date" IS NULL OR "effective_end_date" > "effective_start_date"),
    CONSTRAINT "chk_detention_policies_caps_positive" CHECK (("max_charge_per_stop" IS NULL OR "max_charge_per_stop" > 0) AND ("max_charge_per_day" IS NULL OR "max_charge_per_day" > 0) AND ("max_charge_per_shipment" IS NULL OR "max_charge_per_shipment" > 0)),
    CONSTRAINT "chk_detention_policies_max_minutes" CHECK ("max_billable_minutes_per_stop" IS NULL OR "max_billable_minutes_per_stop" > 0),
    CONSTRAINT "chk_detention_policies_layover_pair" CHECK ("convert_to_layover_at_minutes" IS NULL OR "layover_accessorial_charge_id" IS NOT NULL),
    CONSTRAINT "chk_detention_policies_notice_gate" CHECK ("notification_requirement" <> 'Required' OR "unnotified_behavior" <> 'Bill'),
    CONSTRAINT "chk_detention_policies_org_default_scope" CHECK (NOT "is_org_default" OR ("customer_id" IS NULL AND "location_id" IS NULL AND "shipment_type_ids" IS NULL AND "service_type_ids" IS NULL AND "commodity_ids" IS NULL AND "stop_types" IS NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_policies_code" ON "detention_policies" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_policies_org_default" ON "detention_policies" ("organization_id", "business_unit_id")WHERE
    "is_org_default";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_policies_resolution" ON "detention_policies" ("organization_id", "business_unit_id", "priority" DESC, "specificity_score" DESC, "created_at")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_policies_customer" ON "detention_policies" ("customer_id")WHERE
    "customer_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_policies_location" ON "detention_policies" ("location_id")WHERE
    "location_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "detention_policy_tiers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "detention_policy_id" TEXT NOT NULL,
    "from_minute" INTEGER NOT NULL DEFAULT 0,
    "to_minute" INTEGER,
    "rate" NUMERIC NOT NULL DEFAULT 0,
    "rate_unit" TEXT NOT NULL DEFAULT 'Hour',
    "label" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_detention_policy_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_policy_tiers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policy_tiers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policy_tiers_policy" FOREIGN KEY ("detention_policy_id", "organization_id", "business_unit_id") REFERENCES "detention_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_detention_policy_tiers_band" CHECK ("from_minute" >= 0 AND ("to_minute" IS NULL OR "to_minute" > "from_minute")),
    CONSTRAINT "chk_detention_policy_tiers_rate" CHECK ("rate" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_policy_tiers_policy" ON "detention_policy_tiers" ("detention_policy_id", "from_minute");

--bun:split

CREATE TABLE IF NOT EXISTS "detention_occurrences"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "stop_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "location_id" TEXT NOT NULL,
    "detention_policy_id" TEXT,
    "policy_snapshot" TEXT,
    "calculation_trace" TEXT,
    "stop_type" TEXT NOT NULL,
    "schedule_type" TEXT NOT NULL,
    "appointment_start" INTEGER,
    "appointment_end" INTEGER,
    "arrived_at" INTEGER,
    "departed_at" INTEGER,
    "clock_start_at" INTEGER NOT NULL,
    "clock_stop_at" INTEGER,
    "free_time_expires_at" INTEGER NOT NULL,
    "notice_due_at" INTEGER,
    "notice_deadline_at" INTEGER,
    "is_open" INTEGER NOT NULL DEFAULT 1,
    "arrived_late" INTEGER NOT NULL DEFAULT 0,
    "late_by_minutes" INTEGER NOT NULL DEFAULT 0,
    "free_minutes_granted" INTEGER NOT NULL DEFAULT 0,
    "raw_dwell_minutes" INTEGER NOT NULL DEFAULT 0,
    "billable_minutes" INTEGER NOT NULL DEFAULT 0,
    "rounded_minutes" INTEGER NOT NULL DEFAULT 0,
    "billable_units" NUMERIC NOT NULL DEFAULT 0,
    "gross_amount" NUMERIC NOT NULL DEFAULT 0,
    "billable_amount" NUMERIC NOT NULL DEFAULT 0,
    "driver_pay_minutes" INTEGER NOT NULL DEFAULT 0,
    "driver_pay_amount" NUMERIC NOT NULL DEFAULT 0,
    "net_margin" NUMERIC NOT NULL DEFAULT 0,
    "cap_applied" TEXT NOT NULL DEFAULT 'None',
    "converted_to_layover" INTEGER NOT NULL DEFAULT 0,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "status" TEXT NOT NULL DEFAULT 'Accruing',
    "notification_status" TEXT NOT NULL DEFAULT 'NotRequired',
    "notice_sent_at" INTEGER,
    "suppressed_by_gate" INTEGER NOT NULL DEFAULT 0,
    "requires_approval" INTEGER NOT NULL DEFAULT 0,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "waiver_reason" TEXT,
    "waiver_note" TEXT,
    "waived_by_id" TEXT,
    "waived_at" INTEGER,
    "waived_amount" NUMERIC NOT NULL DEFAULT 0,
    "dispute_note" TEXT,
    "disputed_at" INTEGER,
    "collectability_score" INTEGER NOT NULL DEFAULT 0,
    "evidence_head" TEXT,
    "additional_charge_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_detention_occurrences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_occurrences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_occurrences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_occurrences_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_occurrences_policy" FOREIGN KEY ("detention_policy_id", "organization_id", "business_unit_id") REFERENCES "detention_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_detention_occurrences_clock" CHECK ("clock_stop_at" IS NULL OR "clock_stop_at" >= "clock_start_at"),
    CONSTRAINT "chk_detention_occurrences_minutes" CHECK ("billable_minutes" >= 0 AND "rounded_minutes" >= 0 AND "raw_dwell_minutes" >= 0),
    CONSTRAINT "chk_detention_occurrences_waiver" CHECK ("status" <> 'Waived' OR "waiver_reason" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_occurrences_stop" ON "detention_occurrences" ("organization_id", "business_unit_id", "stop_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_open" ON "detention_occurrences" ("organization_id", "business_unit_id", "free_time_expires_at")WHERE
    "is_open";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_notice_due" ON "detention_occurrences" ("organization_id", "business_unit_id", "notice_due_at")WHERE
    "notification_status" = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_shipment" ON "detention_occurrences" ("shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_status" ON "detention_occurrences" ("organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_location" ON "detention_occurrences" ("location_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_customer" ON "detention_occurrences" ("customer_id", "created_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "detention_evidence"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "detention_occurrence_id" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL DEFAULT 0,
    "kind" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "summary" TEXT NOT NULL,
    "observed_at" INTEGER NOT NULL,
    "recorded_at" INTEGER NOT NULL,
    "recorded_by_id" TEXT,
    "document_id" TEXT,
    "payload" TEXT,
    "prev_hash" TEXT NOT NULL,
    "hash" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_detention_evidence" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_evidence_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_evidence_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_evidence_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_evidence_sequence" ON "detention_evidence" ("detention_occurrence_id", "sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_evidence_occurrence" ON "detention_evidence" ("detention_occurrence_id", "sequence");

--bun:split

CREATE TABLE IF NOT EXISTS "detention_notices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "detention_occurrence_id" TEXT NOT NULL,
    "thread_key" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "channel" TEXT NOT NULL DEFAULT 'Email',
    "delivery_status" TEXT NOT NULL DEFAULT 'Queued',
    "recipients" TEXT,
    "subject" TEXT NOT NULL,
    "body" TEXT NOT NULL,
    "scheduled_for" INTEGER NOT NULL,
    "sent_at" INTEGER,
    "delivered_at" INTEGER,
    "opened_at" INTEGER,
    "failed_at" INTEGER,
    "failure_reason" TEXT,
    "provider_message_id" TEXT,
    "sent_by_id" TEXT,
    "was_automatic" INTEGER NOT NULL DEFAULT 0,
    "satisfies_requirement" INTEGER NOT NULL DEFAULT 0,
    "quoted_free_minutes" INTEGER NOT NULL DEFAULT 0,
    "quoted_rate" NUMERIC,
    "quoted_amount" NUMERIC,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_detention_notices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_notices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_notices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_notices_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_notices_occurrence" ON "detention_notices" ("detention_occurrence_id", "created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_notices_due" ON "detention_notices" ("organization_id", "business_unit_id", "scheduled_for")WHERE
    "delivery_status" = 'Queued';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_notices_thread" ON "detention_notices" ("thread_key");

--bun:split

ALTER TABLE "shipment_controls" ADD COLUMN "default_detention_policy_id" TEXT;

--bun:split

ALTER TABLE "shipment_controls" ADD COLUMN "use_detention_policy_engine" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "additional_charges" ADD COLUMN "detention_occurrence_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_detention_occurrence" ON "additional_charges" ("detention_occurrence_id")WHERE
    "detention_occurrence_id" IS NOT NULL;
