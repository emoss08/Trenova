-- Create "table_change_alerts" table
CREATE TABLE
    "table_change_alerts" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "status" character varying NOT NULL DEFAULT 'A',
        "name" character varying NOT NULL,
        "database_action" character varying NOT NULL,
        "source" character varying NOT NULL,
        "table_name" character varying NULL,
        "topic" character varying NULL,
        "description" text NULL,
        "custom_subject" character varying NULL,
        "function_name" character varying NULL,
        "trigger_name" character varying NULL,
        "listener_name" character varying NULL,
        "email_recipients" text NULL,
        "conditional_logic" jsonb NULL,
        "effective_date" timestamptz NULL,
        "expiration_date" timestamptz NULL,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "table_change_alerts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "table_change_alerts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );