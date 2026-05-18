CREATE TYPE edi_partner_setting_data_type_enum AS ENUM(
    'string',
    'number',
    'integer',
    'boolean',
    'decimal',
    'enum',
    'object',
    'array',
    'map',
    'secret',
    'unknown'
);

CREATE TYPE edi_partner_setting_status_enum AS ENUM(
    'Active',
    'Deprecated',
    'Future'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_partner_setting_schemas"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100),
    "organization_id" varchar(100),
    "document_type_id" varchar(100),
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL,
    "schema_version" bigint NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "status" edi_partner_setting_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_partner_setting_schemas" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_partner_setting_schemas_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_scope"
    ON "edi_partner_setting_schemas"(
        COALESCE("organization_id", ''),
        COALESCE("business_unit_id", ''),
        COALESCE("document_type_id", ''),
        "standard",
        "transaction_set",
        "direction",
        "x12_version",
        "schema_version"
    );

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_lookup"
    ON "edi_partner_setting_schemas"("standard", "transaction_set", "direction", "x12_version", "status", "schema_version" DESC);

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_document_type"
    ON "edi_partner_setting_schemas"("document_type_id", "standard", "transaction_set", "direction", "x12_version", "status", "schema_version" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_partner_setting_fields"(
    "id" varchar(100) NOT NULL,
    "schema_id" varchar(100) NOT NULL,
    "path" text NOT NULL,
    "label" varchar(200) NOT NULL,
    "description" text,
    "data_type" edi_partner_setting_data_type_enum NOT NULL,
    "required" boolean NOT NULL DEFAULT FALSE,
    "nullable" boolean NOT NULL DEFAULT FALSE,
    "default_value" jsonb,
    "allowed_values" jsonb NOT NULL DEFAULT '[]',
    "secret" boolean NOT NULL DEFAULT FALSE,
    "group_key" varchar(100),
    "display_order" integer NOT NULL DEFAULT 0,
    "validation_pattern" text,
    "min_length" integer NOT NULL DEFAULT 0,
    "max_length" integer NOT NULL DEFAULT 0,
    "usage_notes" text,
    "status" edi_partner_setting_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_partner_setting_fields" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_partner_setting_fields_schema" FOREIGN KEY ("schema_id") REFERENCES "edi_partner_setting_schemas"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_path"
    ON "edi_partner_setting_fields"("schema_id", "path");

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_group"
    ON "edi_partner_setting_fields"("schema_id", "group_key", "status");

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_required"
    ON "edi_partner_setting_fields"("schema_id", "required", "status");

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_secret"
    ON "edi_partner_setting_fields"("schema_id", "secret", "status");

--bun:split
ALTER TABLE "edi_partner_document_profiles"
    ADD COLUMN IF NOT EXISTS "partner_settings_schema_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "partner_settings_schema_version" bigint;

ALTER TABLE "edi_partner_document_profiles"
    ADD CONSTRAINT "fk_edi_partner_document_profiles_partner_settings_schema" FOREIGN KEY ("partner_settings_schema_id") REFERENCES "edi_partner_setting_schemas"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS "idx_edi_partner_document_profiles_partner_settings_schema"
    ON "edi_partner_document_profiles"("partner_settings_schema_id", "partner_settings_schema_version");

--bun:split
INSERT INTO "edi_partner_setting_schemas"(
    "id",
    "business_unit_id",
    "organization_id",
    "document_type_id",
    "standard",
    "transaction_set",
    "direction",
    "x12_version",
    "schema_version",
    "name",
    "description",
    "status"
)
VALUES (
    'edips_x12_204_out_partner_v1',
    NULL,
    NULL,
    NULL,
    'X12',
    '204',
    'Outbound',
    '004010',
    1,
    'X12 204 Outbound Partner Settings',
    'Global partner settings metadata for outbound X12 204 load tender profiles.',
    'Active'
)
ON CONFLICT DO NOTHING;

WITH schema_row AS (
    SELECT "id" FROM "edi_partner_setting_schemas"
    WHERE "id" = 'edips_x12_204_out_partner_v1'
),
fields("id", "path", "label", "description", "data_type", "required", "nullable", "default_value", "allowed_values", "secret", "group_key", "display_order", "validation_pattern", "min_length", "max_length", "usage_notes", "status") AS (
    VALUES
        ('edipsf_x12_204_carrier_scac', 'carrier.scac', 'Carrier SCAC', 'Carrier Standard Carrier Alpha Code used in B2 and partner-specific references.', 'string', TRUE, FALSE, NULL, '[]'::jsonb, FALSE, 'carrier', 10, '^[A-Za-z0-9]{2,4}$', 2, 4, 'Required for active outbound 204 profiles.', 'Active'),
        ('edipsf_x12_204_carrier_name', 'carrier.name', 'Carrier Name', 'Carrier display name for partner-specific output.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'carrier', 20, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_carrier_code', 'carrier.code', 'Carrier Code', 'Partner-specific carrier code used outside SCAC-qualified elements.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'carrier', 30, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_contact_name', 'contact.name', 'Contact Name', 'Primary EDI tender contact name.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'contact', 40, NULL, 0, 60, NULL, 'Active'),
        ('edipsf_x12_204_contact_phone', 'contact.phone', 'Contact Phone', 'Primary EDI tender contact phone.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'contact', 50, '^[0-9+(). -]{7,25}$', 0, 25, NULL, 'Active'),
        ('edipsf_x12_204_contact_email', 'contact.email', 'Contact Email', 'Primary EDI tender contact email.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'contact', 60, '^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$', 0, 120, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_code', 'billTo.code', 'Bill-To Code', 'Partner bill-to code for billing party references.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 70, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_name', 'billTo.name', 'Bill-To Name', 'Partner bill-to name override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 80, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_address_1', 'billTo.addressLine1', 'Bill-To Address Line 1', 'Partner bill-to address line 1 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 90, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_address_2', 'billTo.addressLine2', 'Bill-To Address Line 2', 'Partner bill-to address line 2 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 100, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_city', 'billTo.city', 'Bill-To City', 'Partner bill-to city override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 110, NULL, 0, 50, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_state', 'billTo.stateCode', 'Bill-To State Code', 'Partner bill-to state code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 120, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_postal', 'billTo.postalCode', 'Bill-To Postal Code', 'Partner bill-to postal code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 130, NULL, 0, 15, NULL, 'Active'),
        ('edipsf_x12_204_bill_to_country', 'billTo.countryCode', 'Bill-To Country Code', 'Partner bill-to country code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'billTo', 140, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_shipper_code', 'shipper.code', 'Shipper Code', 'Partner shipper code for N1 loops.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 150, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_shipper_name', 'shipper.name', 'Shipper Name', 'Partner shipper name override for N1 loops.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 160, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_shipper_address_1', 'shipper.addressLine1', 'Shipper Address Line 1', 'Partner shipper address line 1 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 170, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_shipper_address_2', 'shipper.addressLine2', 'Shipper Address Line 2', 'Partner shipper address line 2 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 180, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_shipper_city', 'shipper.city', 'Shipper City', 'Partner shipper city override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 190, NULL, 0, 50, NULL, 'Active'),
        ('edipsf_x12_204_shipper_state', 'shipper.stateCode', 'Shipper State Code', 'Partner shipper state code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 200, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_shipper_postal', 'shipper.postalCode', 'Shipper Postal Code', 'Partner shipper postal code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 210, NULL, 0, 15, NULL, 'Active'),
        ('edipsf_x12_204_shipper_country', 'shipper.countryCode', 'Shipper Country Code', 'Partner shipper country code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'shipper', 220, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_consignee_code', 'consignee.code', 'Consignee Code', 'Partner consignee code for N1 loops.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 230, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_consignee_name', 'consignee.name', 'Consignee Name', 'Partner consignee name override for N1 loops.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 240, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_consignee_address_1', 'consignee.addressLine1', 'Consignee Address Line 1', 'Partner consignee address line 1 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 250, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_consignee_address_2', 'consignee.addressLine2', 'Consignee Address Line 2', 'Partner consignee address line 2 override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 260, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_consignee_city', 'consignee.city', 'Consignee City', 'Partner consignee city override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 270, NULL, 0, 50, NULL, 'Active'),
        ('edipsf_x12_204_consignee_state', 'consignee.stateCode', 'Consignee State Code', 'Partner consignee state code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 280, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_consignee_postal', 'consignee.postalCode', 'Consignee Postal Code', 'Partner consignee postal code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 290, NULL, 0, 15, NULL, 'Active'),
        ('edipsf_x12_204_consignee_country', 'consignee.countryCode', 'Consignee Country Code', 'Partner consignee country code override.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'consignee', 300, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_default_equipment_type', 'defaultEquipmentType', 'Default Equipment Type', 'Partner default equipment type when shipment equipment is unavailable.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'defaults', 310, NULL, 0, 40, NULL, 'Active'),
        ('edipsf_x12_204_default_payment_method', 'defaultPaymentMethod', 'Default Payment Method', 'Default shipment payment method qualifier.', 'enum', FALSE, TRUE, to_jsonb('PP'::text), '["CC","PP","TP"]'::jsonb, FALSE, 'defaults', 320, NULL, 0, 2, NULL, 'Active'),
        ('edipsf_x12_204_default_weight_unit', 'defaultWeightUnit', 'Default Weight Unit', 'Default X12 weight unit code.', 'enum', FALSE, TRUE, to_jsonb('L'::text), '["L","K"]'::jsonb, FALSE, 'defaults', 330, NULL, 0, 1, NULL, 'Active'),
        ('edipsf_x12_204_default_packaging_code', 'defaultPackagingCode', 'Default Packaging Code', 'Default L5 packaging code.', 'enum', FALSE, TRUE, to_jsonb('PCS'::text), '["PCS","PLT","CTN","BOX","SKD"]'::jsonb, FALSE, 'defaults', 340, NULL, 0, 5, NULL, 'Active'),
        ('edipsf_x12_204_reference_bol_qualifier', 'referenceQualifiers.bol', 'BOL Qualifier', 'Qualifier used for bill of lading references.', 'string', FALSE, TRUE, to_jsonb('BM'::text), '[]'::jsonb, FALSE, 'referenceQualifiers', 350, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_reference_po_qualifier', 'referenceQualifiers.purchaseOrder', 'Purchase Order Qualifier', 'Qualifier used for purchase order references.', 'string', FALSE, TRUE, to_jsonb('PO'::text), '[]'::jsonb, FALSE, 'referenceQualifiers', 360, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_reference_customer_qualifier', 'referenceQualifiers.customerReference', 'Customer Reference Qualifier', 'Qualifier used for customer shipment references.', 'string', FALSE, TRUE, to_jsonb('CR'::text), '[]'::jsonb, FALSE, 'referenceQualifiers', 370, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_reference_pickup_qualifier', 'referenceQualifiers.pickup', 'Pickup Reference Qualifier', 'Qualifier used for pickup references.', 'string', FALSE, TRUE, to_jsonb('PU'::text), '[]'::jsonb, FALSE, 'referenceQualifiers', 380, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_reference_delivery_qualifier', 'referenceQualifiers.delivery', 'Delivery Reference Qualifier', 'Qualifier used for delivery references.', 'string', FALSE, TRUE, to_jsonb('DO'::text), '[]'::jsonb, FALSE, 'referenceQualifiers', 390, NULL, 0, 3, NULL, 'Active'),
        ('edipsf_x12_204_stop_pickup_reason', 'stopReasonMappings.pickup', 'Pickup Stop Reason', 'S5 reason code for pickup stops.', 'string', FALSE, TRUE, to_jsonb('LD'::text), '[]'::jsonb, FALSE, 'stopReasonMappings', 400, NULL, 0, 2, NULL, 'Active'),
        ('edipsf_x12_204_stop_delivery_reason', 'stopReasonMappings.delivery', 'Delivery Stop Reason', 'S5 reason code for delivery stops.', 'string', FALSE, TRUE, to_jsonb('UL'::text), '[]'::jsonb, FALSE, 'stopReasonMappings', 410, NULL, 0, 2, NULL, 'Active'),
        ('edipsf_x12_204_accessorial_code_map', 'accessorialCodes.codeMap', 'Accessorial Code Map', 'Partner accessorial code map keyed by source accessorial code.', 'map', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'accessorialCodes', 420, NULL, 0, 0, NULL, 'Active'),
        ('edipsf_x12_204_accessorial_default_code', 'accessorialCodes.defaultCode', 'Default Accessorial Code', 'Partner default accessorial code when no explicit mapping is available.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'accessorialCodes', 430, NULL, 0, 10, NULL, 'Active'),
        ('edipsf_x12_204_commodity_description', 'commodityDefaults.description', 'Default Commodity Description', 'Default commodity description when shipment commodity text is unavailable.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'commodityDefaults', 440, NULL, 0, 80, NULL, 'Active'),
        ('edipsf_x12_204_commodity_nmfc', 'commodityDefaults.nmfcCode', 'Default NMFC Code', 'Default NMFC code for commodity details.', 'string', FALSE, TRUE, NULL, '[]'::jsonb, FALSE, 'commodityDefaults', 450, NULL, 0, 20, NULL, 'Active'),
        ('edipsf_x12_204_envelope_sender_qualifier', 'envelope.senderQualifier', 'ISA Sender Qualifier', 'Partner-specific ISA sender qualifier override.', 'string', FALSE, TRUE, to_jsonb('ZZ'::text), '[]'::jsonb, FALSE, 'envelope', 460, NULL, 0, 2, NULL, 'Active'),
        ('edipsf_x12_204_envelope_receiver_qualifier', 'envelope.receiverQualifier', 'ISA Receiver Qualifier', 'Partner-specific ISA receiver qualifier override.', 'string', FALSE, TRUE, to_jsonb('ZZ'::text), '[]'::jsonb, FALSE, 'envelope', 470, NULL, 0, 2, NULL, 'Active'),
        ('edipsf_x12_204_envelope_usage_indicator', 'envelope.usageIndicator', 'Usage Indicator', 'ISA usage indicator override.', 'enum', FALSE, TRUE, to_jsonb('T'::text), '["T","P"]'::jsonb, FALSE, 'envelope', 480, NULL, 0, 1, NULL, 'Active')
)
INSERT INTO "edi_partner_setting_fields"(
    "id",
    "schema_id",
    "path",
    "label",
    "description",
    "data_type",
    "required",
    "nullable",
    "default_value",
    "allowed_values",
    "secret",
    "group_key",
    "display_order",
    "validation_pattern",
    "min_length",
    "max_length",
    "usage_notes",
    "status"
)
SELECT
    fields."id",
    schema_row."id",
    fields."path",
    fields."label",
    fields."description",
    fields."data_type"::edi_partner_setting_data_type_enum,
    fields."required",
    fields."nullable",
    fields."default_value",
    fields."allowed_values",
    fields."secret",
    fields."group_key",
    fields."display_order",
    fields."validation_pattern",
    fields."min_length",
    fields."max_length",
    fields."usage_notes",
    fields."status"::edi_partner_setting_status_enum
FROM fields
CROSS JOIN schema_row
ON CONFLICT DO NOTHING;
