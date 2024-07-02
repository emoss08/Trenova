CREATE TABLE
    IF NOT EXISTS "table_change_alerts"
(
    "created_at"       TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "id"               uuid                 NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum          NOT NULL DEFAULT 'Active',
    "name"             VARCHAR(50)          NOT NULL,
    "database_action"  database_action_enum NOT NULL,
    "topic_name"       VARCHAR(200)         NOT NULL,
    "description"      TEXT,
    "custom_subject"   VARCHAR,
    "delivery_method"  delivery_method_enum NOT NULL DEFAULT 'Email',
    "email_recipients" TEXT,
    "effective_date"   date,
    "expiration_date"  date,
    "business_unit_id" uuid                 NOT NULL,
    "organization_id"  uuid                 NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "table_change_alerts_name_organization_id_unq" ON "table_change_alerts" (LOWER("name"), organization_id);
