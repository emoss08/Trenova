-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
    "postal_code" us_postal_code NOT NULL,
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
CREATE INDEX "idx_workers_business_unit" ON "workers"("business_unit_id");

--bun:split
CREATE INDEX "idx_workers_organization" ON "workers"("organization_id");

--bun:split
CREATE INDEX "idx_workers_state" ON "workers"("state_id")
WHERE
    state_id IS NOT NULL;

--bun:split
CREATE INDEX "idx_workers_status" ON "workers"("status");

--bun:split
CREATE INDEX "idx_workers_type" ON "workers"("type");

--bun:split
CREATE INDEX "idx_workers_name" ON "workers"("last_name", "first_name");

--bun:split
CREATE INDEX "idx_workers_created_updated" ON "workers"("created_at", "updated_at");

--bun:split
CREATE INDEX "idx_workers_org_bu" ON "workers"("organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE workers IS 'Stores information about company workers (employees and contractors)';

--bun:split
ALTER TABLE "workers"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
ALTER TABLE "workers"
    ADD COLUMN IF NOT EXISTS whole_name varchar(201) GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_workers_search ON workers USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION workers_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.first_name, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.last_name, '')), 'A') || setweight(to_tsvector('english', COALESCE(CAST(NEW.status AS text), '')), 'B');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS workers_search_vector_trigger ON workers;

--bun:split
CREATE TRIGGER workers_search_vector_trigger
    BEFORE INSERT OR UPDATE ON workers
    FOR EACH ROW
    EXECUTE FUNCTION workers_search_vector_update();

--bun:split
ALTER TABLE workers
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE workers
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE workers
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_workers_trgm_name ON workers USING gin((first_name || ' ' || last_name) gin_trgm_ops);

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

