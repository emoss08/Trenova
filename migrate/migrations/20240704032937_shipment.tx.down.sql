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

DROP TABLE IF EXISTS "customers" CASCADE;

-- bun:split

DROP TYPE IF EXISTS shipment_status_enum CASCADE;
DROP TYPE IF EXISTS rating_method_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "shipments_pro_number_organization_id_unq" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_status" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_ship_date" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_bill_date" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_customer_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_origin_location_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_destination_location_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_created_by_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_business_unit_id" CASCADE;
