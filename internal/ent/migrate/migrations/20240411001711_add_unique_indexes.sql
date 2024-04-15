-- Add a new index for the `accessorial_charges` table
DROP INDEX IF EXISTS accessorialcharge_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_accessorialcharge_code_organization_id ON accessorial_charges (LOWER(code), organization_id);

-- Add a new index for the `charge_types` table
DROP INDEX IF EXISTS chargetype_name_organization_id CASCADE;

CREATE UNIQUE INDEX unq_chargetype_name_organization_id ON charge_types (LOWER(name), organization_id);

-- Add a new index for the `comment_types` table
DROP INDEX IF EXISTS commenttype_name_organization_id CASCADE;

CREATE UNIQUE INDEX unq_commenttype_name_organization_id ON comment_types (LOWER(name), organization_id);

-- Add a new index for the `customers` table
DROP INDEX IF EXISTS customer_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_customer_code_organization_id ON customers (LOWER(code), organization_id);

-- Add a new index for the `document_classifications` table
CREATE UNIQUE INDEX unq_documentclass_code_organization_id ON document_classifications (LOWER(code), organization_id);

-- Add a new index for the `delay_codes` table
DROP INDEX IF EXISTS delaycode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_delaycode_code_organization_id ON delay_codes (LOWER(code), organization_id);

-- Add a new index for the `division_codes` table
DROP INDEX IF EXISTS divisioncode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_divisioncode_code_organization_id ON division_codes (LOWER(code), organization_id);

-- Add a new index for the `equipment_manufactuers` table
DROP INDEX IF EXISTS equipmentmanufactuer_name_organization_id CASCADE;

CREATE UNIQUE INDEX unq_equipmentmanufactuer_name_organization_id ON equipment_manufactuers (LOWER(name), organization_id);

-- Add a new index for the `equipment_types` table
DROP INDEX IF EXISTS equipmenttype_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_equipmenttype_code_organization_id ON equipment_types (LOWER(code), organization_id);

-- Add a new index for the `equipment_types` table
DROP INDEX IF EXISTS fleetcode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_fleetcode_code_organization_id ON fleet_codes (LOWER(code), organization_id);

-- Add a new index for the `general_ledger_accounts` table
DROP INDEX IF EXISTS generalledgeraccount_account_number_organization_id CASCADE;

CREATE UNIQUE INDEX unq_generalledgeraccount_account_number_organization_id ON general_ledger_accounts (LOWER(account_number), organization_id);

-- Add a new index for the `location_categories` table
DROP INDEX IF EXISTS locationcategory_name_organization_id CASCADE;

CREATE UNIQUE INDEX unq_locationcategory_name_organization_id ON location_categories (LOWER(name), organization_id);

-- Add a new index for the `fleet_codes` table
DROP INDEX IF EXISTS unq_fleetcode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_fleetcode_code_organization_id ON fleet_codes (LOWER(code), organization_id);

-- Add a new index for the `qualifier_codes` table
DROP INDEX IF EXISTS qualifiercode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_qualifiercode_code_organization_id ON qualifier_codes (LOWER(code), organization_id);

-- Add a new index for the `reason_codes` table
DROP INDEX IF EXISTS reasoncode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_reasoncode_code_organization_id ON reason_codes (LOWER(code), organization_id);

-- Add a new index for the `revenue_codes` table
DROP INDEX IF EXISTS revenuecode_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_revenuecode_code_organization_id ON revenue_codes (LOWER(code), organization_id);

-- Add a new index for the `service_types` table
DROP INDEX IF EXISTS servicetype_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_servicetype_code_organization_id ON service_types (LOWER(code), organization_id);

-- Add a new index for the `table_change_alerts` table
CREATE UNIQUE INDEX unq_tablechangealert_name_organization_id ON table_change_alerts (LOWER(name), organization_id);

-- Add a new index for the `trailers` table
DROP INDEX IF EXISTS trailer_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_trailer_code_organization_id ON trailers (LOWER(code), organization_id);

-- Add a new index for the `tractors` table
DROP INDEX IF EXISTS tractor_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_tractor_code_organization_id ON tractors (LOWER(code), organization_id);

-- Add a new index for the `locations` table
DROP INDEX IF EXISTS location_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_location_code_organization_id ON locations (LOWER(code), organization_id);

-- Add a new index for the `shipments` table
DROP INDEX IF EXISTS shipment_pro_number_organization_id CASCADE;

CREATE UNIQUE INDEX unq_shipment_pro_number_organization_id ON shipments (LOWER(pro_number), organization_id);

-- Add a new index for the `shipment_types` table
DROP INDEX IF EXISTS shipmenttype_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_shipmenttype_code_organization_id ON shipment_types (LOWER(code), organization_id);

-- Add a new index for the `tags` table
DROP INDEX IF EXISTS tag_name_organization_id CASCADE;

CREATE UNIQUE INDEX unq_tag_name_organization_id ON tags (LOWER(name), organization_id);

-- Add a new index for the `tractors` table
DROP INDEX IF EXISTS unq_tractor_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_tractor_code_organization_id ON tractors (LOWER(code), organization_id);

-- Add a new index for the `trailers` table
DROP INDEX IF EXISTS unq_trailer_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_trailer_code_organization_id ON trailers (LOWER(code), organization_id);

-- Add a new index for the `workers` table
DROP INDEX IF EXISTS worker_code_organization_id CASCADE;

CREATE UNIQUE INDEX unq_worker_code_organization_id ON workers (LOWER(code), organization_id);