DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'worker_type_enum') THEN
            CREATE TYPE worker_type_enum AS ENUM ('Employee', 'Contractor');
        END IF;
    END
$$;

-- bun:split

CREATE TABLE
    IF NOT EXISTS "workers"
(
    "id"               uuid             NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid             NOT NULL,
    "organization_id"  uuid             NOT NULL,
    "status"           status_enum      NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10)      NOT NULL,
    "worker_type"      worker_type_enum NOT NULL DEFAULT 'Employee',
    "first_name"       VARCHAR(255)     NOT NULL,
    "last_name"        VARCHAR(255)     NOT NULL,
    "address_line_1"   VARCHAR(150),
    "address_line_2"   VARCHAR(150),
    "city"             VARCHAR(150),
    "postal_code"      VARCHAR(10),
    "state_id"         uuid,
    "fleet_code_id"    uuid,
    "manager_id"       uuid,
    "external_id"      VARCHAR(255),
    "version"          bigint           NOT NULL,
    "created_at"       TIMESTAMPTZ      NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ      NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("fleet_code_id") REFERENCES fleet_codes ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("manager_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "workers_code_organization_id_unq" ON "workers" (LOWER("code"), organization_id);
CREATE INDEX idx_workers_name ON workers (code);
CREATE INDEX idx_workers_org_bu ON workers (organization_id, business_unit_id);
CREATE INDEX idx_workers_created_at ON workers (created_at);

--bun:split

COMMENT ON COLUMN workers.id IS 'Unique identifier for the worker, generated as a UUID';
COMMENT ON COLUMN workers.business_unit_id IS 'Foreign key referencing the business unit that this worker belongs to';
COMMENT ON COLUMN workers.organization_id IS 'Foreign key referencing the organization that this worker belongs to';
COMMENT ON COLUMN workers.status IS 'The current status of the worker, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN workers.code IS 'A unique code for identifying the worker, limited to 10 characters';
COMMENT ON COLUMN workers.worker_type IS 'The type of worker, using the worker_type_enum (e.g., Employee, Contractor)';
COMMENT ON COLUMN workers.first_name IS 'The first name of the worker';
COMMENT ON COLUMN workers.last_name IS 'The last name of the worker';
COMMENT ON COLUMN workers.address_line_1 IS 'The first line of the worker''s address, limited to 150 characters';
COMMENT ON COLUMN workers.address_line_2 IS 'The second line of the worker''s address, limited to 150 characters';
COMMENT ON COLUMN workers.city IS 'The city of the worker''s address, limited to 150 characters';
COMMENT ON COLUMN workers.postal_code IS 'The postal code of the worker''s address, limited to 10 characters';
COMMENT ON COLUMN workers.state_id IS 'Foreign key referencing the state of the worker''s address';
COMMENT ON COLUMN workers.fleet_code_id IS 'Foreign key referencing the fleet code that this worker belongs to';
COMMENT ON COLUMN workers.manager_id IS 'Foreign key referencing the user that is the manager of this worker';
COMMENT ON COLUMN workers.external_id IS 'An external identifier for the worker, limited to 255 characters';
COMMENT ON COLUMN workers.created_at IS 'Timestamp of when the worker was created, defaults to the current timestamp';
COMMENT ON COLUMN workers.updated_at IS 'Timestamp of the last update to the worker, defaults to the current timestamp';

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'worker_endorsement_enum') THEN
            CREATE TYPE worker_endorsement_enum AS ENUM ('None', 'Tanker', 'Hazmat', 'TankerHazmat');
        END IF;
    END
$$;

--bun:split

CREATE TABLE
    IF NOT EXISTS "worker_profiles"
(
    "id"                      uuid                    NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"        uuid                    NOT NULL,
    "organization_id"         uuid                    NOT NULL,
    "worker_id"               uuid                    NOT NULL,
    "license_number"          VARCHAR(50)             NOT NULL,
    "state_id"                uuid                    NOT NULL,
    "date_of_birth"           DATE,
    "license_expiration_date" DATE,
    "hazmat_expiration_date"  DATE,
    "hire_date"               DATE,
    "termination_date"        DATE,
    "physical_due_date"       DATE,
    "mvr_due_date"            DATE,
    "endorsements"            worker_endorsement_enum NOT NULL DEFAULT 'None',
    "version"                 BIGINT                  NOT NULL,
    "created_at"              TIMESTAMPTZ             NOT NULL DEFAULT current_timestamp,
    "updated_at"              TIMESTAMPTZ             NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("worker_id") REFERENCES workers ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    UNIQUE ("worker_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "worker_profiles_license_number_organization_id_unq" ON "worker_profiles" (LOWER("license_number"), organization_id);
CREATE INDEX idx_worker_profiles_license_number ON worker_profiles (license_number);
CREATE INDEX idx_worker_profiles_org_bu ON worker_profiles (organization_id, business_unit_id);
CREATE INDEX idx_worker_profiles_created_at ON worker_profiles (created_at);

--bun:split

COMMENT ON COLUMN worker_profiles.id IS 'Unique identifier for the worker profile, generated as a UUID';
COMMENT ON COLUMN worker_profiles.business_unit_id IS 'Foreign key referencing the business unit that this worker profile belongs to';
COMMENT ON COLUMN worker_profiles.organization_id IS 'Foreign key referencing the organization that this worker profile belongs to';
COMMENT ON COLUMN worker_profiles.worker_id IS 'Foreign key referencing the worker that this profile belongs to';
COMMENT ON COLUMN worker_profiles.license_number IS 'The license number of the worker, limited to 50 characters';
COMMENT ON COLUMN worker_profiles.state_id IS 'Foreign key referencing the state of the worker''s license';
COMMENT ON COLUMN worker_profiles.date_of_birth IS 'The date of birth of the worker';
COMMENT ON COLUMN worker_profiles.license_expiration_date IS 'The expiration date of the worker''s license';
COMMENT ON COLUMN worker_profiles.hazmat_expiration_date IS 'The expiration date of the worker''s hazmat endorsement';
COMMENT ON COLUMN worker_profiles.hire_date IS 'The date that the worker was hired';
COMMENT ON COLUMN worker_profiles.termination_date IS 'The date that the worker was terminated';
COMMENT ON COLUMN worker_profiles.physical_due_date IS 'The due date for the worker''s physical';
COMMENT ON COLUMN worker_profiles.mvr_due_date IS 'The due date for the worker''s MVR';
COMMENT ON COLUMN worker_profiles.endorsements IS 'The endorsements that the worker has, using the worker_endorsement_enum (e.g., None, Tanker, Hazmat, TankerHazmat)';
COMMENT ON COLUMN worker_profiles.created_at IS 'Timestamp of when the worker profile was created, defaults to the current timestamp';
COMMENT ON COLUMN worker_profiles.updated_at IS 'Timestamp of the last update to the worker profile, defaults to the current timestamp';

