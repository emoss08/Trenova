CREATE TABLE IF NOT EXISTS "fleet_codes" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "manager_id" varchar(100),
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "description" text,
    "revenue_goal" numeric(10, 2),
    "deadhead_goal" numeric(10, 2),
    "mileage_goal" numeric(10, 2),
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_fleet_codes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fleet_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_manager" FOREIGN KEY ("manager_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
-- Indexes for fleet_codes table
CREATE INDEX "idx_fleet_codes_business_unit" ON "fleet_codes" ("business_unit_id");

CREATE INDEX "idx_fleet_codes_organization" ON "fleet_codes" ("organization_id");

CREATE INDEX "idx_fleet_codes_manager" ON "fleet_codes" ("manager_id")
WHERE
    manager_id IS NOT NULL;

CREATE INDEX "idx_fleet_codes_color" ON "fleet_codes" ("color");

CREATE INDEX "idx_fleet_codes_created_updated" ON "fleet_codes" ("created_at", "updated_at");

COMMENT ON TABLE "fleet_codes" IS 'Stores information about fleet codes';

--bun:split
--alter the workers table to add the fleet_code_id column
ALTER TABLE "workers"
    ADD COLUMN "fleet_code_id" varchar(100);

--add the foreign key constraint
ALTER TABLE "workers"
    ADD CONSTRAINT "fk_workers_fleet_code" FOREIGN KEY ("fleet_code_id", "organization_id", "business_unit_id") REFERENCES "fleet_codes" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

