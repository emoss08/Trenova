-- Copyright (c) 2024 Trenova Technologies, LLC
--
-- Licensed under the Business Source License 1.1 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     https://trenova.app/pricing/
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--
-- Key Terms:
-- - Non-production use only
-- - Change Date: 2026-11-16
-- - Change License: GNU General Public License v2 or later
--
-- For full license text, see the LICENSE file in the root directory.

CREATE TABLE IF NOT EXISTS "tractor_assignments"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "tractor_id"       uuid        NOT NULL,
    "shipment_id"      uuid        NOT NULL,
    "shipment_move_id" uuid        NOT NULL,
    "assigned_by_id"   uuid        NOT NULL,
    "sequence"         integer     NOT NULL,
    "assigned_at"      TIMESTAMPTZ NOT NULL,
    "completed_at"     TIMESTAMPTZ,
    "status"           varchar(20) NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("shipment_id") REFERENCES shipments ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("shipment_move_id") REFERENCES shipment_moves ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("tractor_id") REFERENCES tractors ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("assigned_by_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    UNIQUE ("shipment_move_id", "tractor_id"),
    UNIQUE ("tractor_id", "sequence")
);

--bun:split
CREATE INDEX idx_tractor_assignments_org_bu ON tractor_assignments (organization_id, business_unit_id);
CREATE INDEX idx_tractor_assignments_shipment ON tractor_assignments (shipment_id);
CREATE INDEX idx_tractor_assignments_shipment_move ON tractor_assignments (shipment_move_id);
CREATE INDEX idx_tractor_assignments_tractor ON tractor_assignments (tractor_id);

--bun:split
COMMENT ON COLUMN tractor_assignments.id IS 'Unique identifier for the tractor assignment, generated as a UUID';
COMMENT ON COLUMN tractor_assignments.business_unit_id IS 'Foreign key referencing the business unit that this tractor assignment belongs to';
COMMENT ON COLUMN tractor_assignments.organization_id IS 'Foreign key referencing the organization that this tractor assignment belongs to';
COMMENT ON COLUMN tractor_assignments.tractor_id IS 'Foreign key referencing the tractor that is assigned to the shipment move';
COMMENT ON COLUMN tractor_assignments.shipment_id IS 'Foreign key referencing the shipment that the tractor is assigned to';
COMMENT ON COLUMN tractor_assignments.shipment_move_id IS 'Foreign key referencing the shipment move that the tractor is assigned to';
COMMENT ON COLUMN tractor_assignments.sequence IS 'The sequence number of the tractor assignment';
COMMENT ON COLUMN tractor_assignments.assigned_at IS 'Timestamp of when the tractor was assigned to the shipment move';
COMMENT ON COLUMN tractor_assignments.completed_at IS 'Timestamp of when the tractor assignment was completed';
COMMENT ON COLUMN tractor_assignments.status IS 'The current status of the tractor assignment';