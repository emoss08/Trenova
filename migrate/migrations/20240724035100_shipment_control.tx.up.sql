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

CREATE TABLE IF NOT EXISTS "shipment_control"
(
    "id"                         uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid NOT NULL,
    "organization_id" uuid NOT NULL,
    "enforce_rev_code"           BOOLEAN     NOT NULL DEFAULT FALSE,
    "enforce_voided_comm"        BOOLEAN     NOT NULL DEFAULT FALSE,
    "auto_total_shipment"        BOOLEAN     NOT NULL DEFAULT FALSE,
    "compare_origin_destination" BOOLEAN     NOT NULL DEFAULT FALSE,
    "check_for_duplicate_bol"   BOOLEAN     NOT NULL DEFAULT FALSE,
    "created_at"                 TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"                 TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_control_org_bu" ON "shipment_control" (organization_id, business_unit_id);

CREATE UNIQUE INDEX IF NOT EXISTS "shipment_control_org_id_unq" ON "shipment_control" (organization_id);

-- bun:split
COMMENT ON COLUMN "shipment_control"."enforce_rev_code" IS 'Requires a revenue code to be entered for each shipment';
COMMENT ON COLUMN "shipment_control"."enforce_voided_comm" IS 'Requires a comment to be entered when voiding a shipment';
COMMENT ON COLUMN "shipment_control"."auto_total_shipment" IS 'Automatically calculates the total shipment billing total';
COMMENT ON COLUMN "shipment_control"."compare_origin_destination" IS 'Ensures the origin and destination are different';
COMMENT ON COLUMN "shipment_control"."check_for_duplicate_bol" IS 'Checks for duplicate BOL numbers';