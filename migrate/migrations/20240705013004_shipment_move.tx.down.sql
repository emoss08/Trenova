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

DROP TABLE IF EXISTS "shipment_moves" CASCADE;

-- bun:split

DROP TYPE IF EXISTS shipment_move_status_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "idx_shipment_move_shipment_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_tractor_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_trailer_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_primary_worker_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_secondary_worker_id" CASCADE;
DROP INDEX IF EXISTS "idx_customer_org_bu" CASCADE;
DROP INDEX IF EXISTS "idx_customer_created_at" CASCADE;
