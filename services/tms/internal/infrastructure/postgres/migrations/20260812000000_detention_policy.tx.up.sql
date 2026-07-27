CREATE TYPE "detention_policy_status_enum" AS ENUM(
    'Active',
    'Inactive',
    'Draft'
);

CREATE TYPE "detention_clock_start_basis_enum" AS ENUM(
    'Arrival',
    'Appointment',
    'LaterOfArrivalOrAppointment',
    'EarlierOfArrivalOrAppointment'
);

CREATE TYPE "detention_late_arrival_rule_enum" AS ENUM(
    'NoEffect',
    'Forfeit',
    'ClockFromAppointment',
    'ReduceFreeTime'
);

CREATE TYPE "detention_rounding_mode_enum" AS ENUM(
    'Up',
    'Down',
    'Nearest',
    'Exact'
);

CREATE TYPE "detention_rate_source_enum" AS ENUM(
    'Accessorial',
    'Tiers'
);

CREATE TYPE "detention_tier_rate_unit_enum" AS ENUM(
    'Hour',
    'Day',
    'Flat'
);

CREATE TYPE "detention_notification_requirement_enum" AS ENUM(
    'None',
    'Advisory',
    'Required'
);

CREATE TYPE "detention_unnotified_behavior_enum" AS ENUM(
    'Bill',
    'Flag',
    'Suppress'
);

CREATE TYPE "detention_cap_scope_enum" AS ENUM(
    'PerStop',
    'PerDay',
    'PerShipment'
);

CREATE TYPE "detention_cap_kind_enum" AS ENUM(
    'None',
    'MaxBillableMinutes',
    'MaxChargePerStop',
    'MaxChargePerDay',
    'MaxChargePerShipment',
    'LayoverBoundary'
);

CREATE TYPE "detention_occurrence_status_enum" AS ENUM(
    'Accruing',
    'Pending',
    'Approved',
    'Billed',
    'Waived',
    'Disputed',
    'NotBillable'
);

CREATE TYPE "detention_notification_status_enum" AS ENUM(
    'NotRequired',
    'Pending',
    'Sent',
    'Late',
    'Missed',
    'Failed'
);

CREATE TYPE "detention_waiver_reason_enum" AS ENUM(
    'Weather',
    'FacilityClosure',
    'CarrierFault',
    'EquipmentIssue',
    'CustomerGoodwill',
    'DataCorrection',
    'ForceMajeure',
    'Other'
);

CREATE TYPE "detention_evidence_kind_enum" AS ENUM(
    'Arrival',
    'Departure',
    'Appointment',
    'Notice',
    'Document',
    'StatusChange',
    'Recalculated',
    'Waiver',
    'Dispute'
);

CREATE TYPE "detention_evidence_source_enum" AS ENUM(
    'Telematics',
    'Geofence',
    'DriverApp',
    'EDI',
    'Document',
    'Manual',
    'System'
);

CREATE TYPE "detention_notice_kind_enum" AS ENUM(
    'Warning',
    'Started',
    'Update',
    'Final'
);

CREATE TYPE "detention_notice_channel_enum" AS ENUM(
    'Email',
    'EDI',
    'Portal',
    'Manual'
);

CREATE TYPE "detention_notice_delivery_status_enum" AS ENUM(
    'Queued',
    'Sent',
    'Delivered',
    'Opened',
    'Bounced',
    'Failed'
);

--bun:split
CREATE TABLE IF NOT EXISTS "detention_policies"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "description" text,
    "status" detention_policy_status_enum NOT NULL DEFAULT 'Draft',
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "priority" smallint NOT NULL DEFAULT 0,
    "specificity_score" integer NOT NULL DEFAULT 0,
    "customer_id" varchar(100),
    "location_id" varchar(100),
    "shipment_type_ids" jsonb,
    "service_type_ids" jsonb,
    "commodity_ids" jsonb,
    "stop_types" jsonb,
    "appointment_stops_only" boolean NOT NULL DEFAULT FALSE,
    "effective_start_date" bigint,
    "effective_end_date" bigint,
    "clock_start_basis" detention_clock_start_basis_enum NOT NULL DEFAULT 'LaterOfArrivalOrAppointment',
    "late_arrival_rule" detention_late_arrival_rule_enum NOT NULL DEFAULT 'NoEffect',
    "late_arrival_grace_minutes" smallint NOT NULL DEFAULT 0,
    "billing_free_minutes" integer NOT NULL DEFAULT 120,
    "pickup_free_minutes" integer,
    "delivery_free_minutes" integer,
    "pay_free_minutes" integer,
    "minimum_billable_minutes" integer NOT NULL DEFAULT 0,
    "billing_increment_minutes" smallint NOT NULL DEFAULT 15,
    "rounding_mode" detention_rounding_mode_enum NOT NULL DEFAULT 'Up',
    "rate_source" detention_rate_source_enum NOT NULL DEFAULT 'Accessorial',
    "accessorial_charge_id" varchar(100) NOT NULL,
    "max_billable_minutes_per_stop" integer,
    "max_charge_per_stop" numeric(19, 4),
    "max_charge_per_day" numeric(19, 4),
    "max_charge_per_shipment" numeric(19, 4),
    "day_boundary_mode" detention_cap_scope_enum NOT NULL DEFAULT 'PerStop',
    "convert_to_layover_at_minutes" integer,
    "layover_accessorial_charge_id" varchar(100),
    "notification_requirement" detention_notification_requirement_enum NOT NULL DEFAULT 'None',
    "notification_lead_minutes" smallint NOT NULL DEFAULT 30,
    "notification_deadline_minutes" smallint NOT NULL DEFAULT 0,
    "unnotified_behavior" detention_unnotified_behavior_enum NOT NULL DEFAULT 'Bill',
    "auto_send_notice" boolean NOT NULL DEFAULT FALSE,
    "send_departure_summary" boolean NOT NULL DEFAULT FALSE,
    "require_approval_over_amount" numeric(19, 4),
    "auto_approve_under_amount" numeric(19, 4),
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "comments" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_policies_code" ON "detention_policies"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_policies_org_default" ON "detention_policies"("organization_id", "business_unit_id")
WHERE
    "is_org_default";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_policies_resolution" ON "detention_policies"("organization_id", "business_unit_id", "priority" DESC, "specificity_score" DESC, "created_at")
WHERE
    "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_policies_customer" ON "detention_policies"("customer_id")
WHERE
    "customer_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_policies_location" ON "detention_policies"("location_id")
WHERE
    "location_id" IS NOT NULL;

--bun:split
ALTER TABLE "detention_policies"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_policies_search" ON "detention_policies" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE detention_policies IS 'Contractual detention terms resolved per shipment stop by customer, facility, and freight characteristics';

--bun:split
CREATE TABLE IF NOT EXISTS "detention_policy_tiers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "detention_policy_id" varchar(100) NOT NULL,
    "from_minute" integer NOT NULL DEFAULT 0,
    "to_minute" integer,
    "rate" numeric(19, 4) NOT NULL DEFAULT 0,
    "rate_unit" detention_tier_rate_unit_enum NOT NULL DEFAULT 'Hour',
    "label" varchar(100),
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_detention_policy_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_policy_tiers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policy_tiers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_policy_tiers_policy" FOREIGN KEY ("detention_policy_id", "organization_id", "business_unit_id") REFERENCES "detention_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_detention_policy_tiers_band" CHECK ("from_minute" >= 0 AND ("to_minute" IS NULL OR "to_minute" > "from_minute")),
    CONSTRAINT "chk_detention_policy_tiers_rate" CHECK ("rate" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_policy_tiers_policy" ON "detention_policy_tiers"("detention_policy_id", "from_minute");

--bun:split
CREATE TABLE IF NOT EXISTS "detention_occurrences"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "stop_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "location_id" varchar(100) NOT NULL,
    "detention_policy_id" varchar(100),
    "policy_snapshot" jsonb,
    "calculation_trace" jsonb,
    "stop_type" stop_type_enum NOT NULL,
    "schedule_type" stop_schedule_type_enum NOT NULL,
    "appointment_start" bigint,
    "appointment_end" bigint,
    "arrived_at" bigint,
    "departed_at" bigint,
    "clock_start_at" bigint NOT NULL,
    "clock_stop_at" bigint,
    "free_time_expires_at" bigint NOT NULL,
    "notice_due_at" bigint,
    "notice_deadline_at" bigint,
    "is_open" boolean NOT NULL DEFAULT TRUE,
    "arrived_late" boolean NOT NULL DEFAULT FALSE,
    "late_by_minutes" integer NOT NULL DEFAULT 0,
    "free_minutes_granted" integer NOT NULL DEFAULT 0,
    "raw_dwell_minutes" integer NOT NULL DEFAULT 0,
    "billable_minutes" integer NOT NULL DEFAULT 0,
    "rounded_minutes" integer NOT NULL DEFAULT 0,
    "billable_units" numeric(12, 4) NOT NULL DEFAULT 0,
    "gross_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "billable_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "driver_pay_minutes" integer NOT NULL DEFAULT 0,
    "driver_pay_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "net_margin" numeric(19, 4) NOT NULL DEFAULT 0,
    "cap_applied" detention_cap_kind_enum NOT NULL DEFAULT 'None',
    "converted_to_layover" boolean NOT NULL DEFAULT FALSE,
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "status" detention_occurrence_status_enum NOT NULL DEFAULT 'Accruing',
    "notification_status" detention_notification_status_enum NOT NULL DEFAULT 'NotRequired',
    "notice_sent_at" bigint,
    "suppressed_by_gate" boolean NOT NULL DEFAULT FALSE,
    "requires_approval" boolean NOT NULL DEFAULT FALSE,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    "waiver_reason" detention_waiver_reason_enum,
    "waiver_note" text,
    "waived_by_id" varchar(100),
    "waived_at" bigint,
    "waived_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "dispute_note" text,
    "disputed_at" bigint,
    "collectability_score" smallint NOT NULL DEFAULT 0,
    "evidence_head" varchar(64),
    "additional_charge_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_occurrences_stop" ON "detention_occurrences"("organization_id", "business_unit_id", "stop_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_open" ON "detention_occurrences"("organization_id", "business_unit_id", "free_time_expires_at")
WHERE
    "is_open";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_notice_due" ON "detention_occurrences"("organization_id", "business_unit_id", "notice_due_at")
WHERE
    "notification_status" = 'Pending';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_shipment" ON "detention_occurrences"("shipment_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_status" ON "detention_occurrences"("organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_location" ON "detention_occurrences"("location_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_customer" ON "detention_occurrences"("customer_id", "created_at" DESC);

--bun:split
COMMENT ON TABLE detention_occurrences IS 'Immutable per-stop detention events carrying the policy snapshot and calculation trace that produced the charge';

--bun:split
CREATE TABLE IF NOT EXISTS "detention_evidence"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "detention_occurrence_id" varchar(100) NOT NULL,
    "sequence" integer NOT NULL DEFAULT 0,
    "kind" detention_evidence_kind_enum NOT NULL,
    "source" detention_evidence_source_enum NOT NULL,
    "summary" text NOT NULL,
    "observed_at" bigint NOT NULL,
    "recorded_at" bigint NOT NULL,
    "recorded_by_id" varchar(100),
    "document_id" varchar(100),
    "payload" jsonb,
    "prev_hash" varchar(64) NOT NULL,
    "hash" varchar(64) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_detention_evidence" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_evidence_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_evidence_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_evidence_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_detention_evidence_sequence" ON "detention_evidence"("detention_occurrence_id", "sequence");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_evidence_occurrence" ON "detention_evidence"("detention_occurrence_id", "sequence");

--bun:split
COMMENT ON TABLE detention_evidence IS 'Append-only hash-chained evidence trail proving a detention claim was not edited after the fact';

--bun:split
CREATE TABLE IF NOT EXISTS "detention_notices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "detention_occurrence_id" varchar(100) NOT NULL,
    "thread_key" varchar(120) NOT NULL,
    "kind" detention_notice_kind_enum NOT NULL,
    "channel" detention_notice_channel_enum NOT NULL DEFAULT 'Email',
    "delivery_status" detention_notice_delivery_status_enum NOT NULL DEFAULT 'Queued',
    "recipients" jsonb,
    "subject" varchar(255) NOT NULL,
    "body" text NOT NULL,
    "scheduled_for" bigint NOT NULL,
    "sent_at" bigint,
    "delivered_at" bigint,
    "opened_at" bigint,
    "failed_at" bigint,
    "failure_reason" text,
    "provider_message_id" varchar(255),
    "sent_by_id" varchar(100),
    "was_automatic" boolean NOT NULL DEFAULT FALSE,
    "satisfies_requirement" boolean NOT NULL DEFAULT FALSE,
    "quoted_free_minutes" integer NOT NULL DEFAULT 0,
    "quoted_rate" numeric(19, 4),
    "quoted_amount" numeric(19, 4),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_detention_notices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_detention_notices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_notices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_detention_notices_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_notices_occurrence" ON "detention_notices"("detention_occurrence_id", "created_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_notices_due" ON "detention_notices"("organization_id", "business_unit_id", "scheduled_for")
WHERE
    "delivery_status" = 'Queued';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_notices_thread" ON "detention_notices"("thread_key");

--bun:split
COMMENT ON TABLE detention_notices IS 'Customer detention notices with delivery receipts; the evidence that makes a detention charge collectable';

--bun:split
ALTER TABLE "shipment_controls"
    ADD COLUMN IF NOT EXISTS "default_detention_policy_id" varchar(100);

--bun:split
ALTER TABLE "shipment_controls"
    ADD CONSTRAINT "fk_shipment_controls_default_detention_policy" FOREIGN KEY ("default_detention_policy_id", "organization_id", "business_unit_id") REFERENCES "detention_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipment_controls"
    ADD COLUMN IF NOT EXISTS "use_detention_policy_engine" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN shipment_controls.use_detention_policy_engine IS 'Per-org rollout flag: when false the legacy threshold-based detention path remains active';

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "detention_billing_enabled",
    DROP COLUMN IF EXISTS "detention_free_minutes",
    DROP COLUMN IF EXISTS "detention_rate_per_hour",
    DROP COLUMN IF EXISTS "count_detention_only_on_appointment_stops";

--bun:split
ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "detention_occurrence_id" varchar(100);

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "fk_additional_charges_detention_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_additional_charges_detention_occurrence" ON "additional_charges"("detention_occurrence_id")
WHERE
    "detention_occurrence_id" IS NOT NULL;
