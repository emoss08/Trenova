-- Add a new index for the `accessorial_charges` table
DROP INDEX IF EXISTS accessorialcharge_code_organization_id CASCADE;

CREATE UNIQUE INDEX accessorialcharge_code_organization_id ON accessorial_charges (LOWER(code), organization_id);

-- Add a new index for the `charge_types` table
DROP INDEX IF EXISTS chargetype_name_organization_id CASCADE;

CREATE UNIQUE INDEX chargetype_name_organization_id ON charge_types (LOWER(name), organization_id);

-- Add a new index for the `comment_types` table
DROP INDEX IF EXISTS commenttype_name_organization_id CASCADE;

CREATE UNIQUE INDEX commenttype_name_organization_id ON comment_types (LOWER(name), organization_id);

-- Add a new index for the `customers` table
DROP INDEX IF EXISTS customer_code_organization_id CASCADE;

CREATE UNIQUE INDEX customer_code_organization_id ON customers (LOWER(code), organization_id);

-- Add a new index for the `document_classifications` table
CREATE UNIQUE INDEX documentclass_name_organization_id ON document_classifications (LOWER(name), organization_id);

-- Add a new index for the `delay_codes` table
DROP INDEX IF EXISTS delaycode_code_organization_id CASCADE;

CREATE UNIQUE INDEX delaycode_code_organization_id ON delay_codes (LOWER(code), organization_id);

-- Add a new index for the `division_codes` table
DROP INDEX IF EXISTS divisioncode_code_organization_id CASCADE;

CREATE UNIQUE INDEX divisioncode_code_organization_id ON division_codes (LOWER(code), organization_id);

-- Add a new index for the `equipment_manufactuers` table
DROP INDEX IF EXISTS equipmentmanufactuer_name_organization_id CASCADE;

CREATE UNIQUE INDEX equipmentmanufactuer_name_organization_id ON equipment_manufactuers (LOWER(name), organization_id);

-- Add a new index for the `equipment_types` table
DROP INDEX IF EXISTS equipmenttype_name_organization_id CASCADE;

CREATE UNIQUE INDEX equipmenttype_name_organization_id ON equipment_types (LOWER(name), organization_id);

-- Add a new index for the `equipment_types` table
DROP INDEX IF EXISTS fleetcode_code_organization_id CASCADE;

CREATE UNIQUE INDEX fleetcode_code_organization_id ON fleet_codes (LOWER(code), organization_id);

-- Add a new index for the `general_ledger_accounts` table
DROP INDEX IF EXISTS generalledgeraccount_account_number_organization_id CASCADE;

CREATE UNIQUE INDEX generalledgeraccount_account_number_organization_id ON general_ledger_accounts (LOWER(account_number), organization_id);

-- Add a new index for the `location_categories` table
DROP INDEX IF EXISTS locationcategory_name_organization_id CASCADE;

CREATE UNIQUE INDEX locationcategory_name_organization_id ON location_categories (LOWER(name), organization_id);

-- Add a new index for the `fleet_codes` table
DROP INDEX IF EXISTS fleetcode_code_organization_id CASCADE;

CREATE UNIQUE INDEX fleetcode_code_organization_id ON fleet_codes (LOWER(code), organization_id);

-- Add a new index for the `qualifier_codes` table
DROP INDEX IF EXISTS qualifiercode_code_organization_id CASCADE;

CREATE UNIQUE INDEX qualifiercode_code_organization_id ON qualifier_codes (LOWER(code), organization_id);

-- Add a new index for the `reason_codes` table
DROP INDEX IF EXISTS reasoncode_code_organization_id CASCADE;

CREATE UNIQUE INDEX reasoncode_code_organization_id ON reason_codes (LOWER(code), organization_id);

-- Add a new index for the `revenue_codes` table
DROP INDEX IF EXISTS revenuecode_code_organization_id CASCADE;

CREATE UNIQUE INDEX revenuecode_code_organization_id ON revenue_codes (LOWER(code), organization_id);

-- Add a new index for the `service_types` table
DROP INDEX IF EXISTS servicetype_code_organization_id CASCADE;

CREATE UNIQUE INDEX servicetype_code_organization_id ON service_types (LOWER(code), organization_id);

-- Add a new index for the `table_change_alerts` table
CREATE UNIQUE INDEX tablechangealert_name_organization_id ON table_change_alerts (LOWER(name), organization_id);