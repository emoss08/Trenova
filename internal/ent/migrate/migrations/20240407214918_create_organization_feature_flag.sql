-- Create "feature_flags" table
CREATE TABLE "feature_flags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying NOT NULL, "code" character varying(30) NOT NULL, "beta" boolean NOT NULL DEFAULT false, "description" text NOT NULL, "preview_picture_url" character varying NULL, PRIMARY KEY ("id"));
-- Create index "feature_flags_code_key" to table: "feature_flags"
CREATE UNIQUE INDEX "feature_flags_code_key" ON "feature_flags" ("code");
-- Set comment to table: "feature_flags"
COMMENT ON TABLE "feature_flags" IS 'Internal table for storing the feature flags available for Trenova';
-- Create "organization_feature_flags" table
CREATE TABLE "organization_feature_flags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "is_enabled" boolean NOT NULL DEFAULT true, "feature_flag_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "organization_feature_flags_feature_flags_feature_flag" FOREIGN KEY ("feature_flag_id") REFERENCES "feature_flags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "organization_feature_flags_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "organizationfeatureflag_organization_id_feature_flag_id" to table: "organization_feature_flags"
CREATE UNIQUE INDEX "organizationfeatureflag_organization_id_feature_flag_id" ON "organization_feature_flags" ("organization_id", "feature_flag_id");
