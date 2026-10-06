-- The first organization and administrator of a self-hosted install are created
-- by `trenova db bootstrap`, once. This row records that it happened and what it
-- was asked for, so a re-run with the same inputs is a no-op and a re-run with
-- different inputs, or against a database that already has users, is refused.
-- At most one row exists: the singleton column is always true and unique.
CREATE TABLE IF NOT EXISTS "instance_bootstraps"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "admin_user_id" varchar(100),
    "inputs" jsonb NOT NULL,
    "singleton" boolean NOT NULL DEFAULT TRUE,
    "completed_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_instance_bootstraps" PRIMARY KEY ("id"),
    CONSTRAINT "fk_instance_bootstraps_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_instance_bootstraps_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_instance_bootstraps_admin_user" FOREIGN KEY ("admin_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_instance_bootstraps_singleton" CHECK ("singleton"),
    CONSTRAINT "ck_instance_bootstraps_inputs" CHECK (jsonb_typeof("inputs") = 'object'),
    CONSTRAINT "uq_instance_bootstraps_singleton" UNIQUE ("singleton")
);

--bun:split
SELECT
    trenova_rls.reconcile();
