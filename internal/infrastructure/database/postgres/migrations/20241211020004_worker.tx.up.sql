-- Worker type enum with descriptions
CREATE TYPE worker_type_enum AS ENUM(
    'Employee', -- Full-time company employee
    'Contractor' -- Independent contractor
);

--bun:split
CREATE TABLE IF NOT EXISTS "workers"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    -- Core Fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "first_name" varchar(100) NOT NULL,
    "last_name" varchar(100) NOT NULL,
    "type" worker_type_enum NOT NULL DEFAULT 'Employee',
    "profile_pic_url" varchar(255),
    "address_line1" varchar(150) NOT NULL,
    "address_line2" varchar(150),
    "city" varchar(100) NOT NULL,
    "postal_code" varchar(20) NOT NULL,
    "gender" gender_enum NOT NULL,
    "can_be_assigned" boolean NOT NULL DEFAULT FALSE,
    "assignment_blocked" varchar(255),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_workers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_workers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_workers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for workers table
CREATE INDEX "idx_workers_business_unit" ON "workers"("business_unit_id");

CREATE INDEX "idx_workers_organization" ON "workers"("organization_id");

CREATE INDEX "idx_workers_state" ON "workers"("state_id")
WHERE
    state_id IS NOT NULL;

CREATE INDEX "idx_workers_status" ON "workers"("status");

CREATE INDEX "idx_workers_type" ON "workers"("type");

CREATE INDEX "idx_workers_name" ON "workers"("last_name", "first_name");

CREATE INDEX "idx_workers_created_updated" ON "workers"("created_at", "updated_at");

COMMENT ON TABLE workers IS 'Stores information about company workers (employees and contractors)';

--bun:split
-- Endorsement type enum with descriptions
CREATE TYPE endorsement_type_enum AS ENUM(
    'O', -- No endorsements
    'N', -- Tanker vehicles
    'H', -- Hazardous materials
    'X', -- Combination of tanker and hazmat
    'P', -- Passenger vehicles
    'T' -- Double/triple trailers
);

-- Compliance status enum with descriptions
CREATE TYPE compliance_status_enum AS ENUM(
    'Compliant', -- The worker is compliant
    'NonCompliant', -- The worker is non-compliant
    'Pending' -- The worker is pending
);

CREATE TABLE IF NOT EXISTS "worker_profiles"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "license_state_id" varchar(100),
    -- Core Fields
    "dob" bigint NOT NULL,
    "license_number" varchar(50) NOT NULL,
    "endorsement" endorsement_type_enum NOT NULL DEFAULT 'O',
    "hazmat_expiry" bigint,
    "license_expiry" bigint NOT NULL CHECK (license_expiry > 0),
    "hire_date" bigint NOT NULL CHECK (hire_date > 0),
    "termination_date" bigint,
    "physical_due_date" bigint,
    "mvr_due_date" bigint,
    "compliance_status" compliance_status_enum NOT NULL DEFAULT 'Pending',
    "is_qualified" boolean NOT NULL DEFAULT FALSE,
    "disqualification_reason" varchar(255),
    "last_compliance_check" bigint NOT NULL DEFAULT 0,
    "last_mvr_check" bigint NOT NULL DEFAULT 0,
    "last_drug_test" bigint NOT NULL DEFAULT 0,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_worker_profiles" PRIMARY KEY ("id", "worker_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_license_state" FOREIGN KEY ("license_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "check_endorsement_hazmat" CHECK (endorsement NOT IN ('H', 'X') OR hazmat_expiry IS NOT NULL)
);

--bun:split
CREATE INDEX "idx_worker_profiles_unit_org" ON "worker_profiles"("business_unit_id", "organization_id");

CREATE INDEX "idx_worker_profiles_compliance_status" ON "worker_profiles"("compliance_status", "is_qualified");

CREATE INDEX "idx_worker_profiles_dates" ON "worker_profiles"("license_expiry", "hire_date", "termination_date", "physical_due_date", "mvr_due_date")
WHERE
    license_expiry > 0 OR hire_date > 0 OR termination_date > 0 OR physical_due_date > 0 OR mvr_due_date > 0;

CREATE INDEX "idx_worker_profiles_last_checks" ON "worker_profiles"("last_compliance_check", "last_mvr_check", "last_drug_test");

COMMENT ON TABLE worker_profiles IS 'Stores extended worker information including licensing and certification details';

--bun:split
CREATE TYPE worker_pto_status_enum AS ENUM(
    'Requested', -- The PTO request has been requested
    'Approved', -- The PTO request has been approved
    'Rejected', -- The PTO request has been rejected
    'Cancelled' -- The PTO request has been cancelled
);

--bun:split
CREATE TYPE worker_pto_type_enum AS ENUM(
    'Personal', -- Personal leave
    'Vacation', -- Vacation leave
    'Sick', -- Sick leave
    'Holiday', -- Holiday leave
    'Bereavement', -- Bereavement leave
    'Maternity', -- Maternity leave
    'Paternity' -- Paternity leave
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_pto"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "approver_id" varchar(100),
    -- Core Fields
    "status" worker_pto_status_enum NOT NULL DEFAULT 'Requested',
    "type" worker_pto_type_enum NOT NULL DEFAULT 'Vacation',
    "start_date" bigint NOT NULL CHECK (start_date > 0),
    "end_date" bigint NOT NULL CHECK (end_date > 0),
    "reason" varchar(255) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_worker_pto" PRIMARY KEY ("id", "worker_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_approver" FOREIGN KEY ("approver_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_end_date_after_start_date" CHECK ("end_date" > "start_date")
);

-- Indexes
CREATE INDEX "idx_worker_pto_status" ON "worker_pto"("status");

CREATE INDEX "idx_worker_pto_type" ON "worker_pto"("type");

CREATE INDEX "idx_worker_pto_created_updated" ON "worker_pto"("created_at", "updated_at");

-- Composite index to help with overlap validation
CREATE INDEX "idx_worker_pto_worker_dates" ON "worker_pto"("worker_id", "organization_id", "start_date", "end_date");

COMMENT ON TABLE worker_pto IS 'Stores information about a worker''s PTO requests';

--bun:split
CREATE TYPE document_type_enum AS ENUM(
    'MVRs', -- Motor Vehicle Record
    'MedicalCert', -- Medical Certificate
    'CDL', -- Commercial Driver's License
    'ViolationCert', -- Violation Certificate
    'EmploymentHistory', -- Employment History
    'DrugTest', -- Drug Test
    'RoadTest', -- Road Test
    'TrainingCert' -- Training Certificate
);

CREATE TYPE document_requirement_type_enum AS ENUM(
    'Ongoing', -- The document needs to be renewed periodically
    'OneTime', -- The document is collected once
    'Conditional' -- The document is required based on certain conditions
);

CREATE TYPE retention_period_enum AS ENUM(
    '3Years', -- The document retention is for 3 years
    'LifeOfEmployment', -- The document retention is for the duration of employment plus 3 years
    'Custom' -- The document has a custom retention period
);

CREATE TABLE IF NOT EXISTS "document_requirements"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Core Fields
    "name" varchar(255) NOT NULL,
    "description" text,
    "document_type" document_type_enum NOT NULL,
    "requirement_type" document_requirement_type_enum NOT NULL,
    -- CFR Reference
    "cfr_title" varchar(100),
    "cfr_part" varchar(100),
    "cfr_section" varchar(100),
    "cfr_url" varchar(255),
    -- Timing and Retention
    "retention_period" retention_period_enum NOT NULL DEFAULT '3Years',
    "custom_retention_days" int,
    "renewal_period_days" int,
    "reminder_days" int[],
    -- Validation and Requirements
    "is_required" boolean NOT NULL DEFAULT FALSE,
    "validation_rules" jsonb,
    "blocks_assignment" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_document_requirements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_requirements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_requirements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Indexes
CREATE INDEX "idx_document_requirements_business_unit" ON "document_requirements"("business_unit_id");

CREATE INDEX "idx_document_requirements_organization" ON "document_requirements"("organization_id");

CREATE INDEX "idx_document_requirements_created_updated" ON "document_requirements"("created_at", "updated_at");

COMMENT ON TABLE document_requirements IS 'Stores information about document requirements for a business unit';

--bun:split
CREATE TYPE document_status_enum AS ENUM(
    'Pending', -- The document is pending
    'Active', -- The document is active
    'Expired', -- The document is expired
    'Rejected', -- The document is rejected
    'Revoked' -- The document is revoked
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_documents"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "document_requirement_id" varchar(100) NOT NULL,
    -- Core Fields
    "status" document_status_enum NOT NULL DEFAULT 'Pending',
    "file_url" varchar(255) NOT NULL,
    "issue_date" bigint NOT NULL CHECK (issue_date > 0),
    "expiry_date" bigint,
    "validation_data" jsonb,
    "reviewer_id" varchar(100),
    "reviewed_at" bigint,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_worker_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_documents_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_documents_document_requirement" FOREIGN KEY ("document_requirement_id", "organization_id", "business_unit_id") REFERENCES "document_requirements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Indexes
CREATE INDEX "idx_worker_documents_business_unit" ON "worker_documents"("business_unit_id");

CREATE INDEX "idx_worker_documents_organization" ON "worker_documents"("organization_id");

CREATE INDEX "idx_worker_documents_created_updated" ON "worker_documents"("created_at", "updated_at");

CREATE INDEX "idx_worker_documents_status" ON "worker_documents"("status");

CREATE INDEX "idx_worker_documents_document_requirement" ON "worker_documents"("document_requirement_id");

CREATE INDEX "idx_worker_documents_reviewer" ON "worker_documents"("reviewer_id");

CREATE INDEX "idx_worker_documents_dates" ON "worker_documents"("expiry_date", "issue_date", "reviewed_at")
WHERE
    expiry_date IS NOT NULL OR issue_date IS NOT NULL OR reviewed_at IS NOT NULL;

COMMENT ON TABLE worker_documents IS 'Stores information about a worker''s documents';

--bun:split
CREATE TABLE IF NOT EXISTS "document_reviews"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "worker_document_id" varchar(100) NOT NULL,
    "reviewer_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core Fields
    "status" document_status_enum NOT NULL DEFAULT 'Pending',
    "comments" text,
    "reviewed_at" bigint NOT NULL CHECK (reviewed_at > 0),
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_document_reviews" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_reviews_worker_document" FOREIGN KEY ("worker_document_id", "organization_id", "business_unit_id") REFERENCES "worker_documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_reviews_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_reviews_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Indexes
CREATE INDEX "idx_document_reviews_business_unit" ON "document_reviews"("business_unit_id");

CREATE INDEX "idx_document_reviews_organization" ON "document_reviews"("organization_id");

CREATE INDEX "idx_document_reviews_created_updated" ON "document_reviews"("created_at", "updated_at");

CREATE INDEX "idx_document_reviews_status" ON "document_reviews"("status");

CREATE INDEX "idx_document_reviews_reviewed_at" ON "document_reviews"("reviewed_at");

COMMENT ON TABLE document_reviews IS 'Stores information about document reviews';

