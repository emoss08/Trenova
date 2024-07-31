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

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'audit_log_status_enum') THEN CREATE TYPE audit_log_status_enum AS ENUM ('CREATE', 'UPDATE', 'DELETE');

        END IF;

    END
$$;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'log_status_enum') THEN CREATE TYPE log_status_enum AS ENUM ('ATTEMPTED', 'SUCCEEDED', 'FAILED', 'ERROR');

        END IF;

    END
$$;

--bun:split

CREATE TABLE IF NOT EXISTS "audit_logs"
(
    "id"               uuid                  NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid                  NOT NULL,
    "organization_id"  uuid                  NOT NULL,
    "table_name"       varchar(255)          NOT NULL,
    "entity_id"        varchar(255)          NOT NULL,
    "action"           audit_log_status_enum NOT NULL,
    "changes"          jsonb,
    "description"      TEXT,
    "username"         varchar(255)          NOT NULL,
    "error_message"    TEXT,
    "status"           log_status_enum       NOT NULL DEFAULT 'ATTEMPTED',
    "user_id"          uuid                  NOT NULL,
    "timestamp"        TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("user_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_logs_table_name" ON "audit_logs" ("table_name");

CREATE INDEX IF NOT EXISTS "idx_audit_logs_entity_id" ON "audit_logs" ("entity_id");

CREATE INDEX IF NOT EXISTS "idx_audit_logs_user_id" ON "audit_logs" ("user_id");

CREATE INDEX IF NOT EXISTS "idx_audit_logs_username" ON "audit_logs" ("username");

--bun:split

COMMENT ON COLUMN "audit_logs"."id" IS 'Unique identifier for the audit log, generated as a UUID';

COMMENT ON COLUMN "audit_logs".business_unit_id IS 'Foreign key referencing the business unit that this trailer belongs to';

COMMENT ON COLUMN "audit_logs".organization_id IS 'Foreign key referencing the organization that this trailer belongs to';

COMMENT ON COLUMN "audit_logs".table_name IS 'Name of the table that was audited';

COMMENT ON COLUMN "audit_logs".entity_id IS 'Unique identifier of the entity that was audited';

COMMENT ON COLUMN "audit_logs".action IS 'Action that was performed on the entity';

COMMENT ON COLUMN "audit_logs".changes IS 'JSON object containing the changes made to the entity';

COMMENT ON COLUMN "audit_logs".description IS 'Description of the action that was performed';

COMMENT ON COLUMN "audit_logs".username IS 'Username of the user who performed the action';

COMMENT ON COLUMN "audit_logs".error_message IS 'Error message if the action failed';

COMMENT ON COLUMN "audit_logs".status IS 'Status of the action';

COMMENT ON COLUMN "audit_logs".user_id IS 'Foreign key referencing the user who performed the action';

COMMENT ON COLUMN "audit_logs".timestamp IS 'Timestamp when the action was performed';
