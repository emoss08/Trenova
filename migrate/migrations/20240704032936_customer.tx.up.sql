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

CREATE TABLE
    IF NOT EXISTS "customers"
(
    "id"                      uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"        uuid         NOT NULL,
    "organization_id"         uuid         NOT NULL,
    "status"                  status_enum  NOT NULL DEFAULT 'Active',
    "code"                    VARCHAR(10)  NOT NULL,
    "name"                    VARCHAR(150) NOT NULL,
    "address_line_1"          VARCHAR(150) NOT NULL,
    "address_line_2"          VARCHAR(150),
    "city"                    VARCHAR(150) NOT NULL,
    "state_id"                uuid         NOT NULL,
    "has_customer_portal"     BOOLEAN      NOT NULL DEFAULT false,
    "auto_mark_ready_to_bill" BOOLEAN      NOT NULL DEFAULT false,
    "postal_code"             VARCHAR(10)  NOT NULL,
    "version"                 BIGINT       NOT NULL,
    "created_at"              TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"              TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "customers_code_organization_id_unq" ON "customers" (LOWER("code"), organization_id);
CREATE INDEX idx_customer_name ON customers (name);
CREATE INDEX idx_customer_org_bu ON customers (organization_id, business_unit_id);
CREATE INDEX idx_customer_created_at ON customers (created_at);

--bun:split

COMMENT ON COLUMN customers.id IS 'Unique identifier for the customer, generated as a UUID';
COMMENT ON COLUMN customers.business_unit_id IS 'Foreign key referencing the business unit that this customer belongs to';
COMMENT ON COLUMN customers.organization_id IS 'Foreign key referencing the organization that this customer belongs to';
COMMENT ON COLUMN customers.status IS 'The current status of the customer, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN customers.code IS 'A short, unique code for identifying the customer, limited to 10 characters';
COMMENT ON COLUMN customers.name IS 'The name of the customer';
COMMENT ON COLUMN customers.address_line_1 IS 'The first line of the address for the customer';
COMMENT ON COLUMN customers.address_line_2 IS 'The second line of the address for the customer';
COMMENT ON COLUMN customers.city IS 'The city where the customer is located';
COMMENT ON COLUMN customers.postal_code IS 'The postal code for the customer';
COMMENT ON COLUMN customers.state_id IS 'Foreign key referencing the state where the customer is located';
COMMENT ON COLUMN customers.auto_mark_ready_to_bill IS 'A flag indicating whether shipments for this customer should be automatically marked as ready to bill';
COMMENT ON COLUMN customers.created_at IS 'Timestamp of when the customer was created, defaults to the current timestamp';
COMMENT ON COLUMN customers.updated_at IS 'Timestamp of the last update to the customer, defaults to the current timestamp';
