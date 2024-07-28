-- COPYRIGHT(c) 2024 Trenova
--
-- This file is part of Trenova.
--
-- The Trenova software is licensed under the Business Source License 1.1. You are granted the right
-- to copy, modify, and redistribute the software, but only for non-production use or with a total
-- of less than three server instances. Starting from the Change Date (November 16, 2026), the
-- software will be made available under version 2 or later of the GNU General Public License.
-- If you use the software in violation of this license, your rights under the license will be
-- terminated automatically. The software is provided "as is," and the Licensor disclaims all
-- warranties and conditions. If you use this license's text or the "Business Source License" name
-- and trademark, you must comply with the Licensor's covenants, which include specifying the
-- Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
-- Grant, and not modifying the license in any other way.

CREATE TABLE
    IF NOT EXISTS "table_change_alerts" (
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "status" status_enum NOT NULL DEFAULT 'Active',
        "name" VARCHAR(50) NOT NULL,
        "database_action" database_action_enum NOT NULL,
        "topic_name" VARCHAR(200) NOT NULL,
        "description" TEXT,
        "custom_subject" VARCHAR,
        "delivery_method" delivery_method_enum NOT NULL DEFAULT 'Email',
        "email_recipients" TEXT,
        "effective_date" date,
        "expiration_date" date,
        "version" BIGINT NOT NULL,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

CREATE UNIQUE INDEX IF NOT EXISTS "table_change_alerts_name_organization_id_unq" ON "table_change_alerts" (LOWER("name"), organization_id);

CREATE INDEX IF NOT EXISTS idx_table_change_alerts_org_bu ON table_change_alerts (organization_id, business_unit_id);

CREATE INDEX IF NOT EXISTS idx_table_change_alerts_code ON table_change_alerts USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_table_change_alerts_created_at ON table_change_alerts (created_at);

--bun:split
COMMENT ON COLUMN table_change_alerts.id IS 'Unique identifier for the table change alert, generated as a UUID';

COMMENT ON COLUMN table_change_alerts.status IS 'The current status of the table change alert, using the status_enum (e.g., Active, Inactive)';

COMMENT ON COLUMN table_change_alerts.name IS 'A short, unique name for identifying the table change alert, limited to 50 characters';

COMMENT ON COLUMN table_change_alerts.database_action IS 'The database action that triggers the table change alert, using the database_action_enum (e.g., Insert, Update, Delete, All)';

COMMENT ON COLUMN table_change_alerts.topic_name IS 'The name of the topic that the table change alert is associated with, limited to 200 characters';

COMMENT ON COLUMN table_change_alerts.description IS 'A description of the table change alert';

COMMENT ON COLUMN table_change_alerts.custom_subject IS 'A custom subject for the table change alert, limited to 200 characters';

COMMENT ON COLUMN table_change_alerts.delivery_method IS 'The delivery method for the table change alert, using the delivery_method_enum (e.g., Email, Local, Api, Sms)';

COMMENT ON COLUMN table_change_alerts.email_recipients IS 'A list of email recipients for the table change alert';

COMMENT ON COLUMN table_change_alerts.effective_date IS 'The effective date for the table change alert';

COMMENT ON COLUMN table_change_alerts.expiration_date IS 'The expiration date for the table change alert';

COMMENT ON COLUMN table_change_alerts.business_unit_id IS 'Foreign key referencing the business unit that this table change alert belongs to';

COMMENT ON COLUMN table_change_alerts.organization_id IS 'Foreign key referencing the organization that this table change alert belongs to';