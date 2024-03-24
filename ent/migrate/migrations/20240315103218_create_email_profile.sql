-- Create "email_profiles" table
CREATE TABLE
    "email_profiles" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "name" character varying NOT NULL,
        "email" character varying NOT NULL,
        "protocol" character varying NULL,
        "host" character varying NULL,
        "port" bigint NULL,
        "username" character varying NULL,
        "password" character varying NULL,
        "is_default" boolean NOT NULL DEFAULT false,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "email_profiles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "email_profiles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );