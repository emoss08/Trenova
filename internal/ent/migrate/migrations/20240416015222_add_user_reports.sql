-- Create "custom_reports" table
CREATE TABLE
    "custom_reports" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "version" bigint NOT NULL DEFAULT 1,
        "name" character varying NOT NULL,
        "description" character varying NULL,
        "table" character varying NULL,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "custom_reports_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "custom_reports_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

-- Create "user_reports" table
CREATE TABLE
    "user_reports" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "version" bigint NOT NULL DEFAULT 1,
        "report_url" character varying NOT NULL,
        "user_id" uuid NOT NULL,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "user_reports_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "user_reports_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "user_reports_users_reports" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
    );