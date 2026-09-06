--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "employment_verification_status_enum" AS ENUM(
    'Pending',
    'Requested',
    'Received',
    'NoResponse',
    'NotApplicable'
);

CREATE TYPE "employment_verification_method_enum" AS ENUM(
    'Email',
    'Fax',
    'Mail',
    'Phone',
    'Portal',
    'Other'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_employment_verifications"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "employer_name" varchar(150) NOT NULL,
    "employer_dot_number" varchar(20),
    "employer_mc_number" varchar(20),
    "contact_name" varchar(100),
    "contact_phone" varchar(30),
    "contact_email" varchar(150),
    "employed_from" bigint,
    "employed_to" bigint,
    "was_dot_regulated" boolean NOT NULL DEFAULT TRUE,
    "status" employment_verification_status_enum NOT NULL DEFAULT 'Pending',
    "method" employment_verification_method_enum NOT NULL DEFAULT 'Email',
    "requested_at" bigint,
    "response_received_at" bigint,
    "last_follow_up_at" bigint,
    "follow_up_count" integer NOT NULL DEFAULT 0,
    "drug_alcohol_response_received_at" bigint,
    "had_accidents" boolean NOT NULL DEFAULT FALSE,
    "accident_count" integer NOT NULL DEFAULT 0,
    "had_drug_alcohol_violations" boolean NOT NULL DEFAULT FALSE,
    "findings" text,
    "notes" text,
    "document_id" varchar(100),
    "requested_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_employment_verifications" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_employment_verifications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_employment_verifications_period" CHECK ("employed_from" IS NULL OR "employed_to" IS NULL OR "employed_to" >= "employed_from"),
    CONSTRAINT "chk_worker_employment_verifications_received" CHECK (("status" <> 'Received') OR "response_received_at" IS NOT NULL),
    CONSTRAINT "chk_worker_employment_verifications_requested" CHECK ("status" IN ('Pending', 'NotApplicable') OR "requested_at" IS NOT NULL),
    CONSTRAINT "chk_worker_employment_verifications_counts" CHECK ("follow_up_count" >= 0 AND "accident_count" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_employment_verifications_worker" ON "worker_employment_verifications"("worker_id", "employed_to" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_employment_verifications_outstanding" ON "worker_employment_verifications"("organization_id", "business_unit_id", "status")
WHERE
    "status" IN ('Pending', 'Requested');

--bun:split
COMMENT ON TABLE worker_employment_verifications IS 'Safety performance history investigations of a driver''s previous DOT-regulated employers (49 CFR 391.23, 382.413). One row per employer; the follow-up count is the record of good-faith effort when nobody answers.';

--bun:split
ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "driver_qualification_retention_period" integer NOT NULL DEFAULT 1095;

--bun:split
COMMENT ON COLUMN data_retention.driver_qualification_retention_period IS 'Days past termination a driver qualification file is held before it is eligible for purge. 1095 is the three years 49 CFR 391.51(d) requires. Files are only ever flagged, never deleted automatically.';

--bun:split
INSERT INTO "worker_credential_types"("id", "business_unit_id", "organization_id", "code", "name", "description", "category", "is_required", "renewal_window_days", "validity_months", "requires_number", "requires_document", "profile_field", "is_system", "sort_order")
SELECT
    'wct_' || upper(substr(md5(o.id || d.code), 1, 26)),
    o.business_unit_id,
    o.id,
    d.code,
    d.name,
    d.description,
    d.category::worker_credential_category_enum,
    d.is_required,
    d.renewal_window_days,
    d.validity_months,
    FALSE,
    TRUE,
    NULL,
    TRUE,
    d.sort_order
FROM
    "organizations" o
    CROSS JOIN (
        VALUES ('ROAD_TEST', 'Road Test Certificate', 'Road test and certificate, or the licence accepted in its place (49 CFR 391.31, 391.33).', 'Certification', TRUE, 30, NULL::integer, 45),
            ('ANNUAL_REVIEW', 'Annual Review of Driving Record', 'The employer''s annual review of the driver''s record and finding that they remain qualified (49 CFR 391.25(c)).', 'Background', TRUE, 30, 12, 46),
            ('VIOLATION_CERT', 'Annual Violation Certification', 'The driver''s own annual list of traffic convictions, or a signed statement that there were none (49 CFR 391.27).', 'Background', TRUE, 30, 12, 47)) AS d("code", "name", "description", "category", "is_required", "renewal_window_days", "validity_months", "sort_order")
ON CONFLICT
    DO NOTHING;
