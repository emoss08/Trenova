CREATE TYPE "billing_exception_handling_enum" AS ENUM(
    'Queue',
    'Notify',
    'AutoResolve',
    'Reject'
);

--bun:split
CREATE TYPE "approval_requirement_enum" AS ENUM(
    'None',
    'Always',
    'ThresholdBased',
    'ExceptionOnly'
);


--bun:split
CREATE TYPE "transfer_schedule_enum" AS ENUM(
    'Continuous',
    'Hourly',
    'Daily',
    'Weekly'
);

-- Enums with documentation
CREATE TYPE "payment_term_enum" AS ENUM(
    'Net15',
    'Net30',
    'Net45',
    'Net60',
    'Net90',
    'DueOnReceipt'
);

--bun:split
CREATE TYPE "transfer_criteria_enum" AS ENUM(
    'ReadyAndCompleted',
    'Completed',
    'ReadyToBill',
    'DocumentsAttached',
    'PODReceived'
);

--bun:split
CREATE TYPE "auto_bill_criteria_enum" AS ENUM(
    'Delivered',
    'Transferred',
    'MarkedReadyToBill',
    'PODReceived',
    'DocumentsVerified'
);


--bun:split
CREATE TABLE IF NOT EXISTS "billing_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Prefixes for invoice and credit memo numbers
    "invoice_number_prefix" varchar(100) NOT NULL DEFAULT 'INV-',
    "credit_memo_number_prefix" varchar(100) NOT NULL DEFAULT 'CM-',
    -- Invoice Terms
    "payment_term" payment_term_enum NOT NULL DEFAULT 'Net30',
    "show_invoice_due_date" boolean NOT NULL DEFAULT TRUE,
    "invoice_terms" text,
    "invoice_footer" text,
    "show_amount_due" boolean NOT NULL DEFAULT TRUE,
    -- Controls for the billing process
    "auto_transfer" boolean NOT NULL DEFAULT TRUE,
    "transfer_criteria" transfer_criteria_enum NOT NULL DEFAULT 'ReadyAndCompleted',
    "transfer_schedule" transfer_schedule_enum NOT NULL DEFAULT 'Continuous',
    "transfer_batch_size" integer NOT NULL DEFAULT 100 CHECK ("transfer_batch_size" >= 1),
    "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT TRUE,
    -- Enforce customer billing requirements before billing
    "enforce_customer_billing_req" boolean NOT NULL DEFAULT TRUE,
    "validate_customer_rates" boolean NOT NULL DEFAULT TRUE,
    -- Automated billing controls
    "auto_bill" boolean NOT NULL DEFAULT TRUE,
    "auto_bill_criteria" auto_bill_criteria_enum NOT NULL DEFAULT 'Delivered',
    "send_auto_bill_notifications" boolean NOT NULL DEFAULT TRUE,
    "auto_bill_batch_size" integer NOT NULL DEFAULT 100 CHECK ("auto_bill_batch_size" >= 1),
    -- Exception handling
    "billing_exception_handling" billing_exception_handling_enum NOT NULL DEFAULT 'Queue',
    "rate_discrepancy_threshold" numeric(10, 2) NOT NULL DEFAULT 5.00 CHECK ("rate_discrepancy_threshold" >= 0),
    "auto_resolve_minor_discrepancies" boolean NOT NULL DEFAULT TRUE,
    -- Consolidation options
    "allow_invoice_consolidation" boolean NOT NULL DEFAULT TRUE,
    "consolidation_period_days" integer NOT NULL DEFAULT 7 CHECK ("consolidation_period_days" >= 1),
    "group_consolidated_invoices" boolean NOT NULL DEFAULT TRUE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_billing_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one shipment control per organization
    CONSTRAINT "uq_billing_controls_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_billing_controls_business_unit" ON "billing_controls"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_billing_controls_created_at" ON "billing_controls"("created_at", "updated_at");

-- Add comment to describe the table purpose
COMMENT ON TABLE billing_controls IS 'Stores configuration for billing controls and validation rules';

--bun:split
CREATE OR REPLACE FUNCTION billing_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS billing_controls_update_timestamp_trigger ON billing_controls;

CREATE TRIGGER billing_controls_update_timestamp_trigger
    BEFORE UPDATE ON billing_controls
    FOR EACH ROW
    EXECUTE FUNCTION billing_controls_update_timestamp();

--bun:split
ALTER TABLE billing_controls
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE billing_controls
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE billing_controls
    ADD COLUMN IF NOT EXISTS approval_requirement "approval_requirement_enum" NOT NULL DEFAULT 'None';

-- Add comment
COMMENT ON COLUMN billing_controls.approval_requirement IS 'Controls the approval requirement for billing';

