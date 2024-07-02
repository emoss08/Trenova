CREATE TABLE
    IF NOT EXISTS "master_key_generations"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    UNIQUE ("organization_id")
);

-- ================================================
-- bun:split

CREATE TABLE
    IF NOT EXISTS "worker_master_key_generations"
(
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "master_key_id" uuid         NOT NULL,
    "pattern"       VARCHAR(255) NOT NULL,
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- ================================================
-- bun:split

CREATE TABLE
    IF NOT EXISTS "location_master_key_generations"
(
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "master_key_id" uuid         NOT NULL,
    "pattern"       VARCHAR(255) NOT NULL,
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
