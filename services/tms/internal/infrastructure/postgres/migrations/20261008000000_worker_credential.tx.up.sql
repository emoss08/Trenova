--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "worker_credential_category_enum" AS ENUM(
    'License',
    'Medical',
    'Endorsement',
    'Security',
    'Certification',
    'Background',
    'Other'
);

CREATE TYPE "worker_credential_status_enum" AS ENUM(
    'Active',
    'Archived'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_credential_types"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "category" worker_credential_category_enum NOT NULL DEFAULT 'Other',
    "status" status_enum NOT NULL DEFAULT 'Active',
    "is_required" boolean NOT NULL DEFAULT FALSE,
    "required_for_driver_types" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "renewal_window_days" integer NOT NULL DEFAULT 30,
    "validity_months" integer,
    "requires_number" boolean NOT NULL DEFAULT FALSE,
    "requires_document" boolean NOT NULL DEFAULT FALSE,
    "profile_field" varchar(40),
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "sort_order" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_credential_types" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_credential_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credential_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_credential_types_renewal_window" CHECK ("renewal_window_days" >= 0),
    CONSTRAINT "chk_worker_credential_types_validity" CHECK ("validity_months" IS NULL OR "validity_months" > 0)
);
--bun:split
ALTER TABLE "worker_credential_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credential_types_code" ON "worker_credential_types"("organization_id", "business_unit_id", LOWER("code"));
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credential_types_profile_field" ON "worker_credential_types"("organization_id", "business_unit_id", "profile_field")
WHERE
    "profile_field" IS NOT NULL;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_credential_types_status" ON "worker_credential_types"("organization_id", "business_unit_id", "status", "sort_order");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_credential_types_search" ON "worker_credential_types" USING GIN(search_vector);
--bun:split
COMMENT ON TABLE worker_credential_types IS 'Catalog of licences, cards, endorsements and certificates a worker can hold. System rows mirror a worker_profiles column via profile_field.';
--bun:split
CREATE TABLE IF NOT EXISTS "worker_credentials"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "credential_type_id" varchar(100) NOT NULL,
    "status" worker_credential_status_enum NOT NULL DEFAULT 'Active',
    "number" varchar(100),
    "issuing_authority" varchar(100),
    "issued_at" bigint,
    "expires_at" bigint,
    "document_id" varchar(100),
    "notes" text,
    "verified_by_id" varchar(100),
    "verified_at" bigint,
    "archived_by_id" varchar(100),
    "archived_at" bigint,
    "archive_reason" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_credentials" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_credentials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_type" FOREIGN KEY ("credential_type_id", "organization_id", "business_unit_id") REFERENCES "worker_credential_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_worker_credentials_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_credentials_dates" CHECK ("issued_at" IS NULL OR "expires_at" IS NULL OR "expires_at" > "issued_at"),
    CONSTRAINT "chk_worker_credentials_archive" CHECK (("status" = 'Archived') = ("archived_at" IS NOT NULL))
);
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credentials_active_type" ON "worker_credentials"("organization_id", "business_unit_id", "worker_id", "credential_type_id")
WHERE
    "status" = 'Active';
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_credentials_worker" ON "worker_credentials"("worker_id", "status", "expires_at");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_credentials_expiry" ON "worker_credentials"("expires_at")
WHERE
    "status" = 'Active' AND "expires_at" IS NOT NULL;
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_credentials_document" ON "worker_credentials"("document_id")
WHERE
    "document_id" IS NOT NULL;
--bun:split
COMMENT ON TABLE worker_credentials IS 'One row per credential a worker holds. At most one Active row per worker and type; renewals archive the previous row so history is kept.';
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
    d.requires_number,
    d.requires_document,
    d.profile_field,
    TRUE,
    d.sort_order
FROM
    "organizations" o
    CROSS JOIN (
        VALUES ('CDL', 'Commercial Driver''s License', 'State-issued CDL. Number and expiry mirror the worker profile.', 'License', TRUE, 30, NULL::integer, TRUE, TRUE, 'LicenseExpiry', 10),
            ('MED_CARD', 'DOT Medical Card', 'Medical examiner''s certificate (49 CFR 391.43).', 'Medical', TRUE, 30, 24, FALSE, TRUE, 'MedicalCardExpiry', 20),
            ('DOT_PHYSICAL', 'DOT Physical', 'Next physical examination due date.', 'Medical', TRUE, 30, 24, FALSE, FALSE, 'PhysicalDueDate', 30),
            ('MVR', 'Motor Vehicle Record Review', 'Annual MVR pull and review (49 CFR 391.25).', 'Background', TRUE, 30, 12, FALSE, FALSE, 'MVRDueDate', 40),
            ('HAZMAT', 'Hazmat Endorsement', 'H or X endorsement with TSA threat assessment.', 'Endorsement', FALSE, 60, 60, FALSE, TRUE, 'HazmatExpiry', 50),
            ('TWIC', 'TWIC Card', 'Transportation Worker Identification Credential for port access.', 'Security', FALSE, 60, 60, TRUE, TRUE, 'TWICExpiry', 60),
            ('TANKER', 'Tanker Endorsement', 'N endorsement for liquid bulk.', 'Endorsement', FALSE, 30, NULL::integer, FALSE, FALSE, NULL, 70),
            ('DOUBLES', 'Doubles/Triples Endorsement', 'T endorsement for multi-trailer combinations.', 'Endorsement', FALSE, 30, NULL::integer, FALSE, FALSE, NULL, 80),
            ('FORKLIFT', 'Forklift Certification', 'OSHA powered industrial truck certification.', 'Certification', FALSE, 30, 36, FALSE, TRUE, NULL, 90)) AS d("code", "name", "description", "category", "is_required", "renewal_window_days", "validity_months", "requires_number", "requires_document", "profile_field", "sort_order")
ON CONFLICT DO NOTHING;
--bun:split
INSERT INTO "worker_credentials"("id", "business_unit_id", "organization_id", "worker_id", "credential_type_id", "number", "issuing_authority", "expires_at", "created_at", "updated_at")
SELECT
    'wcred_' || upper(substr(md5(wp.worker_id || t.code), 1, 26)),
    wp.business_unit_id,
    wp.organization_id,
    wp.worker_id,
    t.id,
    CASE WHEN t.code = 'CDL' THEN
        wp.license_number
    END,
    CASE WHEN t.code = 'CDL' THEN
        st.abbreviation
    END,
    CASE t.code
    WHEN 'CDL' THEN
        NULLIF(wp.license_expiry, 0)
    WHEN 'MED_CARD' THEN
        wp.medical_card_expiry
    WHEN 'DOT_PHYSICAL' THEN
        wp.physical_due_date
    WHEN 'MVR' THEN
        wp.mvr_due_date
    WHEN 'HAZMAT' THEN
        wp.hazmat_expiry
    WHEN 'TWIC' THEN
        wp.twic_expiry
    END,
    wp.created_at,
    wp.updated_at
FROM
    "worker_profiles" wp
    JOIN "worker_credential_types" t ON t.organization_id = wp.organization_id
        AND t.business_unit_id = wp.business_unit_id
        AND t.profile_field IS NOT NULL
    LEFT JOIN "us_states" st ON st.id = wp.license_state_id
WHERE
    CASE t.code
    WHEN 'CDL' THEN
        NULLIF(wp.license_expiry, 0)
    WHEN 'MED_CARD' THEN
        wp.medical_card_expiry
    WHEN 'DOT_PHYSICAL' THEN
        wp.physical_due_date
    WHEN 'MVR' THEN
        wp.mvr_due_date
    WHEN 'HAZMAT' THEN
        wp.hazmat_expiry
    WHEN 'TWIC' THEN
        wp.twic_expiry
    END IS NOT NULL
ON CONFLICT DO NOTHING;
