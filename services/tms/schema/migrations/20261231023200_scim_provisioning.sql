-- A group mapping hands its role to whoever the identity provider puts in the
-- group, so it records who saved it: provisioning grants the role only while
-- that person may still grant it. A mapping saved before this column existed
-- names nobody and grants nothing until it is saved again.
ALTER TABLE "scim_group_role_mappings"
    ADD COLUMN IF NOT EXISTS "saved_by_id" VARCHAR(100) REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS "scim_users" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "directory_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "user_name" VARCHAR(255) NOT NULL,
    "external_id" VARCHAR(255),
    "active" BOOLEAN NOT NULL DEFAULT TRUE,
    "provisioned" BOOLEAN NOT NULL DEFAULT FALSE,
    "roles_version" BIGINT NOT NULL DEFAULT 0,
    "roles_reconciled_version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_scim_users" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_scim_users_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_users_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_users_directory" FOREIGN KEY ("directory_id") REFERENCES "scim_directories" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_users_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_scim_users_directory_user" UNIQUE ("organization_id", "business_unit_id", "directory_id", "user_id"),
    CONSTRAINT "ck_scim_users_roles_version" CHECK ("roles_reconciled_version" <= "roles_version")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_scim_users_user_name" ON "scim_users" ("organization_id", "business_unit_id", "directory_id", lower("user_name"));

CREATE UNIQUE INDEX IF NOT EXISTS "idx_scim_users_external_id" ON "scim_users" ("organization_id", "business_unit_id", "directory_id", "external_id") WHERE "external_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_scim_users_directory_created" ON "scim_users" ("organization_id", "business_unit_id", "directory_id", "created_at", "id");

CREATE INDEX IF NOT EXISTS "idx_scim_users_roles_pending" ON "scim_users" ("organization_id", "business_unit_id", "directory_id", "id") WHERE "roles_reconciled_version" <> "roles_version";

CREATE INDEX IF NOT EXISTS "idx_scim_users_user" ON "scim_users" ("user_id");

COMMENT ON TABLE "scim_users" IS 'The people a SCIM directory provisioned or linked in its organization; a directory acts only on the people listed here for it';

COMMENT ON COLUMN "scim_users"."provisioned" IS 'True when the directory created the account, false when it linked an account that already belonged to the organization';

COMMENT ON COLUMN "scim_users"."roles_version" IS 'Raised by every change that can move the roles the person''s groups grant; the reconcile workflow brings roles_reconciled_version up to it';

CREATE TABLE IF NOT EXISTS "scim_groups" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "directory_id" VARCHAR(100) NOT NULL,
    "display_name" VARCHAR(255) NOT NULL,
    "external_id" VARCHAR(255),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_scim_groups" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_scim_groups_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_groups_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_groups_directory" FOREIGN KEY ("directory_id") REFERENCES "scim_directories" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_scim_groups_display_name" ON "scim_groups" ("organization_id", "business_unit_id", "directory_id", lower("display_name"));

CREATE UNIQUE INDEX IF NOT EXISTS "idx_scim_groups_external_id" ON "scim_groups" ("organization_id", "business_unit_id", "directory_id", "external_id") WHERE "external_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_scim_groups_directory_created" ON "scim_groups" ("organization_id", "business_unit_id", "directory_id", "created_at", "id");

COMMENT ON TABLE "scim_groups" IS 'Groups a SCIM directory pushed; a group whose external id or display name a group role mapping names hands its members that role';

CREATE TABLE IF NOT EXISTS "scim_group_members" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "directory_id" VARCHAR(100) NOT NULL,
    "group_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_scim_group_members" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_scim_group_members_group" FOREIGN KEY ("group_id", "organization_id", "business_unit_id") REFERENCES "scim_groups" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_scim_group_members_user" FOREIGN KEY ("organization_id", "business_unit_id", "directory_id", "user_id") REFERENCES "scim_users" ("organization_id", "business_unit_id", "directory_id", "user_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_scim_group_members_group_user" UNIQUE ("organization_id", "business_unit_id", "group_id", "user_id")
);

CREATE INDEX IF NOT EXISTS "idx_scim_group_members_user" ON "scim_group_members" ("organization_id", "business_unit_id", "directory_id", "user_id");

CREATE TABLE IF NOT EXISTS "scim_role_grants" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "directory_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "role_id" VARCHAR(100) NOT NULL,
    "assignment_id" VARCHAR(100) NOT NULL,
    "granted_by_id" VARCHAR(100) NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_scim_role_grants" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_scim_role_grants_directory" FOREIGN KEY ("directory_id") REFERENCES "scim_directories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_scim_role_grants_assignment" FOREIGN KEY ("assignment_id") REFERENCES "user_role_assignments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_scim_role_grants_user_role" UNIQUE ("organization_id", "business_unit_id", "directory_id", "user_id", "role_id"),
    CONSTRAINT "uq_scim_role_grants_assignment" UNIQUE ("assignment_id")
);

COMMENT ON TABLE "scim_role_grants" IS 'Role assignments a SCIM directory made from its group mappings; only these are taken away when the groups stop granting the role, never an assignment a person made';

COMMENT ON COLUMN "scim_role_grants"."granted_by_id" IS 'The person who saved the mapping the grant was made under; the grant holds only while they may still grant the role';
