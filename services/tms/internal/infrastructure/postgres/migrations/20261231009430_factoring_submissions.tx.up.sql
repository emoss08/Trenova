--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- How schedules reach a factor: its inbox or its SFTP drop. One row per factor,
-- because the delivery address outlives any one agreement's terms.
CREATE TABLE IF NOT EXISTS "factoring_transports"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "method" varchar(20) NOT NULL DEFAULT 'Email',
    "email_recipients" text[],
    "attach_packets" boolean NOT NULL DEFAULT TRUE,
    "sftp_host" varchar(255),
    "sftp_port" integer,
    "sftp_username" varchar(255),
    "sftp_auth_method" varchar(20),
    "sftp_secret" text,
    "sftp_known_host_key" text,
    "sftp_directory" varchar(500),
    "last_tested_at" bigint,
    "last_test_error" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_transports" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_transports_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_factoring_transports_method" CHECK ("method" IN ('Email', 'Sftp')),
    CONSTRAINT "chk_factoring_transports_auth" CHECK ("sftp_auth_method" IS NULL OR "sftp_auth_method" IN ('Password', 'PrivateKey')),
    CONSTRAINT "chk_factoring_transports_port" CHECK ("sftp_port" IS NULL OR "sftp_port" BETWEEN 1 AND 65535),
    CONSTRAINT "chk_factoring_transports_email" CHECK ("method" <> 'Email' OR cardinality("email_recipients") > 0),
    CONSTRAINT "chk_factoring_transports_sftp" CHECK ("method" <> 'Sftp' OR ("sftp_host" IS NOT NULL AND "sftp_port" IS NOT NULL AND "sftp_username" IS NOT NULL AND "sftp_auth_method" IS NOT NULL AND "sftp_secret" IS NOT NULL AND "sftp_known_host_key" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_transports_factor"
    ON "factoring_transports"("organization_id", "business_unit_id", "factoring_company_id");

--bun:split
-- A schedule of accounts: the invoices sold to a factor in one delivery, with
-- what the agreement in force says each should bring in.
CREATE TABLE IF NOT EXISTS "factoring_submissions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "number" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "agreement_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Sending',
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "invoice_count" integer NOT NULL DEFAULT 0,
    "face_minor" bigint NOT NULL DEFAULT 0,
    "expected_advance_minor" bigint NOT NULL DEFAULT 0,
    "expected_fee_minor" bigint NOT NULL DEFAULT 0,
    "expected_reserve_minor" bigint NOT NULL DEFAULT 0,
    "schedule_pdf_document_id" varchar(100),
    "schedule_csv_document_id" varchar(100),
    "transport_method" varchar(20) NOT NULL,
    "transport_target" varchar(1000) NOT NULL,
    "transport_reference" text,
    "send_attempt_group" integer NOT NULL DEFAULT 1,
    "attempts" integer NOT NULL DEFAULT 0,
    "last_error" text,
    "idempotency_key" varchar(200) NOT NULL,
    "sent_at" bigint,
    "failed_at" bigint,
    "cancelled_at" bigint,
    "cancel_reason" varchar(500),
    "submitted_by_id" varchar(100),
    "agent_run_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_submissions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_submissions_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_submissions_agreement" FOREIGN KEY ("agreement_id", "organization_id", "business_unit_id") REFERENCES "factoring_agreements"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_submissions_status" CHECK ("status" IN ('Sending', 'Sent', 'Failed', 'Cancelled')),
    CONSTRAINT "chk_factoring_submissions_method" CHECK ("transport_method" IN ('Email', 'Sftp')),
    CONSTRAINT "chk_factoring_submissions_totals" CHECK ("invoice_count" > 0 AND "face_minor" >= 0 AND "expected_advance_minor" >= 0 AND "expected_fee_minor" >= 0 AND "expected_reserve_minor" >= 0),
    CONSTRAINT "chk_factoring_submissions_attempts" CHECK ("attempts" >= 0 AND "send_attempt_group" >= 1),
    CONSTRAINT "chk_factoring_submissions_sent" CHECK ("status" <> 'Sent' OR "sent_at" IS NOT NULL),
    CONSTRAINT "chk_factoring_submissions_failed" CHECK ("status" <> 'Failed' OR "failed_at" IS NOT NULL),
    CONSTRAINT "chk_factoring_submissions_cancelled" CHECK (("status" = 'Cancelled') = ("cancelled_at" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submissions_number"
    ON "factoring_submissions"("organization_id", "business_unit_id", "number");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submissions_idempotency"
    ON "factoring_submissions"("organization_id", "business_unit_id", "idempotency_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_factoring_submissions_list"
    ON "factoring_submissions"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_factoring_submissions_factor"
    ON "factoring_submissions"("organization_id", "business_unit_id", "factoring_company_id", "status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_factoring_submissions_failed"
    ON "factoring_submissions"("failed_at")
    WHERE "status" = 'Failed';

--bun:split
-- One invoice on a schedule. Pending while the schedule is live; released when
-- a failed schedule is cancelled, so the invoice can go on another.
CREATE TABLE IF NOT EXISTS "factoring_submission_items"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "submission_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "invoice_number" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "position" integer NOT NULL,
    "face_minor" bigint NOT NULL,
    "expected_advance_minor" bigint NOT NULL DEFAULT 0,
    "expected_fee_minor" bigint NOT NULL DEFAULT 0,
    "expected_reserve_minor" bigint NOT NULL DEFAULT 0,
    "expected_days" integer NOT NULL DEFAULT 0,
    "packet_document_id" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "released_at" bigint,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_submission_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_submission_items_submission" FOREIGN KEY ("submission_id", "organization_id", "business_unit_id") REFERENCES "factoring_submissions"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_submission_items_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_submission_items_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_submission_items_status" CHECK ("status" IN ('Pending', 'Released')),
    CONSTRAINT "chk_factoring_submission_items_released" CHECK (("status" = 'Released') = ("released_at" IS NOT NULL)),
    CONSTRAINT "chk_factoring_submission_items_amounts" CHECK ("face_minor" > 0 AND "expected_advance_minor" >= 0 AND "expected_fee_minor" >= 0 AND "expected_reserve_minor" >= 0 AND "expected_days" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submission_items_live_invoice"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "invoice_id")
    WHERE "status" = 'Pending';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submission_items_position"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "submission_id", "position");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_factoring_submission_items_invoice"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "invoice_id");

--bun:split
ALTER TABLE "factoring_invoice_states"
    DROP CONSTRAINT IF EXISTS "chk_factoring_invoice_states_verdict";

--bun:split
ALTER TABLE "factoring_invoice_states"
    ADD CONSTRAINT "chk_factoring_invoice_states_verdict" CHECK ("verdict" IN ('Ready', 'NotReady', 'Submitted', 'Excluded'));

--bun:split
SELECT trenova_rls.reconcile();
