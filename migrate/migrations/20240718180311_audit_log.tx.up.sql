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

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'log_status_enum') THEN CREATE TYPE log_status_enum AS ENUM ('ATTEMPTED', 'SUCCEEDED', 'FAILED', 'ERROR');

        END IF;

    END
$$;


CREATE TABLE IF NOT EXISTS "audit_logs"
(
    "id"                uuid                  NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"  uuid                  NOT NULL,
    "organization_id"   uuid                  NOT NULL,
    "table_name"        varchar(255)          NOT NULL,
    "entity_id"         varchar(255)          NOT NULL,
    "action"            audit_log_status_enum NOT NULL,
    "data"              jsonb,
    "attempted_changes" jsonb,
    "actual_changes"    jsonb,
    "description"       TEXT,
    "error_message"     TEXT,
    "attempt_id"        uuid,
    "status"            log_status_enum       NOT NULL DEFAULT 'ATTEMPTED',
    "user_id"           uuid                  NOT NULL,
    "timestamp"         TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("user_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);