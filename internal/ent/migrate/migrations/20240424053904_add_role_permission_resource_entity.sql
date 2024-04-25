-- Create "resources" table
CREATE TABLE "resources" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "type" character varying NOT NULL, "description" character varying NULL, PRIMARY KEY ("id"));
-- Create index "resources_type_key" to table: "resources"
CREATE UNIQUE INDEX "resources_type_key" ON "resources" ("type");
-- Create "permissions" table
CREATE TABLE "permissions" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "description" character varying NULL, "action" character varying NULL, "resource" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "permissions_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "permissions_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "permissions"
COMMENT ON COLUMN "permissions" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "permissions"
COMMENT ON COLUMN "permissions" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "permissions"
COMMENT ON COLUMN "permissions" ."version" IS 'The current version of this entity.';
-- Create "roles" table
CREATE TABLE "roles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "description" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "roles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "roles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "roles"
COMMENT ON COLUMN "roles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "roles"
COMMENT ON COLUMN "roles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "roles"
COMMENT ON COLUMN "roles" ."version" IS 'The current version of this entity.';
-- Create "role_permissions" table
CREATE TABLE "role_permissions" ("role_id" uuid NOT NULL, "permission_id" uuid NOT NULL, PRIMARY KEY ("role_id", "permission_id"), CONSTRAINT "role_permissions_permission_id" FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "role_permissions_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "role_users" table
CREATE TABLE "role_users" ("role_id" uuid NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("role_id", "user_id"), CONSTRAINT "role_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "role_users_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
