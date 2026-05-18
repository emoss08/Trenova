CREATE TYPE edi_source_context_data_type_enum AS ENUM(
    'string',
    'number',
    'integer',
    'boolean',
    'timestamp',
    'date',
    'decimal',
    'object',
    'array',
    'unknown'
);

CREATE TYPE edi_source_context_kind_enum AS ENUM(
    'shipment',
    'repeat',
    'partner',
    'runtime',
    'mapping',
    'organization',
    'customer',
    'location',
    'commodity',
    'charge',
    'envelope'
);

CREATE TYPE edi_source_context_field_status_enum AS ENUM(
    'Active',
    'Deprecated',
    'Future'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_source_context_schemas"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100),
    "organization_id" varchar(100),
    "standard" edi_standard_enum NOT NULL,
    "transaction_set" edi_transaction_set_enum NOT NULL,
    "direction" edi_document_direction_enum NOT NULL,
    "x12_version" varchar(20) NOT NULL,
    "context_key" varchar(100) NOT NULL,
    "schema_version" bigint NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "status" edi_source_context_field_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_source_context_schemas" PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_source_context_schemas_scope"
    ON "edi_source_context_schemas"(
        COALESCE("organization_id", ''),
        COALESCE("business_unit_id", ''),
        "standard",
        "transaction_set",
        "direction",
        "x12_version",
        "context_key",
        "schema_version"
    );

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_schemas_lookup"
    ON "edi_source_context_schemas"("standard", "transaction_set", "direction", "x12_version", "context_key", "status");

--bun:split
CREATE TABLE IF NOT EXISTS "edi_source_context_fields"(
    "id" varchar(100) NOT NULL,
    "schema_id" varchar(100) NOT NULL,
    "path" text NOT NULL,
    "source_kind" edi_source_context_kind_enum NOT NULL,
    "data_type" edi_source_context_data_type_enum NOT NULL,
    "repeated" boolean NOT NULL DEFAULT FALSE,
    "repeat_path" text,
    "parent_path" text,
    "display_name" varchar(200) NOT NULL,
    "description" text,
    "status" edi_source_context_field_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_source_context_fields" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_source_context_fields_schema" FOREIGN KEY ("schema_id") REFERENCES "edi_source_context_schemas"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_source_context_fields_path"
    ON "edi_source_context_fields"("schema_id", "path", COALESCE("repeat_path", ''));

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_fields_search"
    ON "edi_source_context_fields"("schema_id", "source_kind", "status", "repeated");

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_fields_prefix"
    ON "edi_source_context_fields"("schema_id", "path");

--bun:split
INSERT INTO "edi_source_context_schemas"(
    "id",
    "business_unit_id",
    "organization_id",
    "standard",
    "transaction_set",
    "direction",
    "x12_version",
    "context_key",
    "schema_version",
    "name",
    "description",
    "status"
)
VALUES (
    'edisc_x12_204_out_load_tender_v1',
    NULL,
    NULL,
    'X12',
    '204',
    'Outbound',
    '004010',
    'loadTender',
    1,
    'X12 204 Outbound Load Tender Source Context',
    'Global source context metadata for outbound X12 204 load tender templates.',
    'Active'
)
ON CONFLICT DO NOTHING;

WITH schema_row AS (
    SELECT "id" FROM "edi_source_context_schemas"
    WHERE "id" = 'edisc_x12_204_out_load_tender_v1'
),
fields("id", "path", "source_kind", "data_type", "repeated", "repeat_path", "parent_path", "display_name", "description", "status") AS (
    VALUES
        ('ediscf_x12_204_shipment_id', 'shipment.shipmentId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Shipment ID', 'Load tender shipment identifier.', 'Active'),
        ('ediscf_x12_204_business_unit_id', 'shipment.businessUnitId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Business Unit ID', 'Source business unit identifier.', 'Active'),
        ('ediscf_x12_204_organization_id', 'shipment.organizationId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Organization ID', 'Source organization identifier.', 'Active'),
        ('ediscf_x12_204_service_type_id', 'shipment.serviceTypeId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Service Type ID', 'Service type identifier.', 'Active'),
        ('ediscf_x12_204_service_type_label', 'shipment.serviceTypeLabel', 'shipment', 'string', FALSE, NULL, 'shipment', 'Service Type Label', 'Service type display label.', 'Active'),
        ('ediscf_x12_204_shipment_type_id', 'shipment.shipmentTypeId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Shipment Type ID', 'Shipment type identifier.', 'Active'),
        ('ediscf_x12_204_shipment_type_label', 'shipment.shipmentTypeLabel', 'shipment', 'string', FALSE, NULL, 'shipment', 'Shipment Type Label', 'Shipment type display label.', 'Active'),
        ('ediscf_x12_204_customer_id', 'shipment.customerId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Customer ID', 'Customer identifier.', 'Active'),
        ('ediscf_x12_204_customer_label', 'shipment.customerLabel', 'shipment', 'string', FALSE, NULL, 'shipment', 'Customer Label', 'Customer display label.', 'Active'),
        ('ediscf_x12_204_formula_template_id', 'shipment.formulaTemplateId', 'shipment', 'string', FALSE, NULL, 'shipment', 'Formula Template ID', 'Rating formula template identifier.', 'Active'),
        ('ediscf_x12_204_formula_template_label', 'shipment.formulaTemplateLabel', 'shipment', 'string', FALSE, NULL, 'shipment', 'Formula Template Label', 'Rating formula template label.', 'Active'),
        ('ediscf_x12_204_bol', 'shipment.bol', 'shipment', 'string', FALSE, NULL, 'shipment', 'Bill of Lading', 'Shipment bill of lading number.', 'Active'),
        ('ediscf_x12_204_pieces', 'shipment.pieces', 'shipment', 'integer', FALSE, NULL, 'shipment', 'Pieces', 'Total shipment pieces.', 'Active'),
        ('ediscf_x12_204_weight', 'shipment.weight', 'shipment', 'integer', FALSE, NULL, 'shipment', 'Weight', 'Total shipment weight.', 'Active'),
        ('ediscf_x12_204_temperature_min', 'shipment.temperatureMin', 'shipment', 'integer', FALSE, NULL, 'shipment', 'Minimum Temperature', 'Minimum temperature requirement.', 'Active'),
        ('ediscf_x12_204_temperature_max', 'shipment.temperatureMax', 'shipment', 'integer', FALSE, NULL, 'shipment', 'Maximum Temperature', 'Maximum temperature requirement.', 'Active'),
        ('ediscf_x12_204_freight_charge_amount', 'shipment.freightChargeAmount', 'shipment', 'decimal', FALSE, NULL, 'shipment', 'Freight Charge Amount', 'Freight charge amount.', 'Active'),
        ('ediscf_x12_204_other_charge_amount', 'shipment.otherChargeAmount', 'shipment', 'decimal', FALSE, NULL, 'shipment', 'Other Charge Amount', 'Other charge amount.', 'Active'),
        ('ediscf_x12_204_base_rate', 'shipment.baseRate', 'shipment', 'decimal', FALSE, NULL, 'shipment', 'Base Rate', 'Base rate amount.', 'Active'),
        ('ediscf_x12_204_total_charge_amount', 'shipment.totalChargeAmount', 'shipment', 'decimal', FALSE, NULL, 'shipment', 'Total Charge Amount', 'Total charge amount.', 'Active'),
        ('ediscf_x12_204_rating_unit', 'shipment.ratingUnit', 'shipment', 'integer', FALSE, NULL, 'shipment', 'Rating Unit', 'Rating unit.', 'Active'),
        ('ediscf_x12_204_rating_detail', 'shipment.ratingDetail', 'shipment', 'object', FALSE, NULL, 'shipment', 'Rating Detail', 'Rendered rating detail object.', 'Active'),
        ('ediscf_x12_204_rating_payment_method', 'shipment.ratingDetail.paymentMethod', 'shipment', 'string', FALSE, NULL, 'shipment.ratingDetail', 'Payment Method', 'Shipment method of payment.', 'Active'),
        ('ediscf_x12_204_rating_note', 'shipment.ratingDetail.note', 'shipment', 'string', FALSE, NULL, 'shipment.ratingDetail', 'Rating Note', 'Shipment rating note.', 'Active'),
        ('ediscf_x12_204_moves', 'shipment.moves', 'shipment', 'array', FALSE, NULL, 'shipment', 'Moves', 'Load tender moves.', 'Active'),
        ('ediscf_x12_204_commodities', 'shipment.commodities', 'shipment', 'array', FALSE, NULL, 'shipment', 'Commodities', 'Load tender commodities.', 'Active'),
        ('ediscf_x12_204_additional_charges', 'shipment.additionalCharges', 'shipment', 'array', FALSE, NULL, 'shipment', 'Additional Charges', 'Load tender additional charges.', 'Active'),
        ('ediscf_x12_204_required_mapping_ids', 'shipment.requiredMappingEntityIds', 'shipment', 'object', FALSE, NULL, 'shipment', 'Required Mapping Entity IDs', 'Entity IDs required for mapping review.', 'Active'),

        ('ediscf_x12_204_stop_location_id', 'repeat.locationId', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Location ID', 'Stop location identifier.', 'Active'),
        ('ediscf_x12_204_stop_location_label', 'repeat.locationLabel', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Location Label', 'Stop location label.', 'Active'),
        ('ediscf_x12_204_stop_location_name', 'repeat.locationName', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Location Name', 'Stop location name.', 'Active'),
        ('ediscf_x12_204_stop_location_code', 'repeat.locationCode', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Location Code', 'Stop location code.', 'Active'),
        ('ediscf_x12_204_stop_address_1', 'repeat.locationAddressLine1', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Address Line 1', 'Stop address line 1.', 'Active'),
        ('ediscf_x12_204_stop_address_2', 'repeat.locationAddressLine2', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Address Line 2', 'Stop address line 2.', 'Active'),
        ('ediscf_x12_204_stop_city', 'repeat.locationCity', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop City', 'Stop city.', 'Active'),
        ('ediscf_x12_204_stop_state', 'repeat.locationStateCode', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop State Code', 'Stop state code.', 'Active'),
        ('ediscf_x12_204_stop_postal', 'repeat.locationPostalCode', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Postal Code', 'Stop postal code.', 'Active'),
        ('ediscf_x12_204_stop_type', 'repeat.type', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Type', 'Stop type.', 'Active'),
        ('ediscf_x12_204_stop_schedule_type', 'repeat.scheduleType', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Schedule Type', 'Stop schedule type.', 'Active'),
        ('ediscf_x12_204_stop_sequence', 'repeat.sequence', 'repeat', 'integer', TRUE, 'moves.0.stops', 'repeat', 'Stop Sequence', 'Stop sequence.', 'Active'),
        ('ediscf_x12_204_stop_pieces', 'repeat.pieces', 'repeat', 'integer', TRUE, 'moves.0.stops', 'repeat', 'Stop Pieces', 'Stop pieces.', 'Active'),
        ('ediscf_x12_204_stop_weight', 'repeat.weight', 'repeat', 'integer', TRUE, 'moves.0.stops', 'repeat', 'Stop Weight', 'Stop weight.', 'Active'),
        ('ediscf_x12_204_stop_window_start', 'repeat.scheduledWindowStart', 'repeat', 'timestamp', TRUE, 'moves.0.stops', 'repeat', 'Scheduled Window Start', 'Stop scheduled window start.', 'Active'),
        ('ediscf_x12_204_stop_window_end', 'repeat.scheduledWindowEnd', 'repeat', 'timestamp', TRUE, 'moves.0.stops', 'repeat', 'Scheduled Window End', 'Stop scheduled window end.', 'Active'),
        ('ediscf_x12_204_stop_address_line', 'repeat.addressLine', 'repeat', 'string', TRUE, 'moves.0.stops', 'repeat', 'Stop Address Line', 'Formatted stop address line.', 'Active'),

        ('ediscf_x12_204_commodity_id', 'repeat.commodityId', 'repeat', 'string', TRUE, 'commodities', 'repeat', 'Commodity ID', 'Commodity identifier.', 'Active'),
        ('ediscf_x12_204_commodity_label', 'repeat.commodityLabel', 'repeat', 'string', TRUE, 'commodities', 'repeat', 'Commodity Label', 'Commodity display label.', 'Active'),
        ('ediscf_x12_204_commodity_name', 'repeat.commodityName', 'repeat', 'string', TRUE, 'commodities', 'repeat', 'Commodity Name', 'Commodity name.', 'Active'),
        ('ediscf_x12_204_commodity_description', 'repeat.commodityDescription', 'repeat', 'string', TRUE, 'commodities', 'repeat', 'Commodity Description', 'Commodity description.', 'Active'),
        ('ediscf_x12_204_commodity_weight', 'repeat.weight', 'repeat', 'integer', TRUE, 'commodities', 'repeat', 'Commodity Weight', 'Commodity weight.', 'Active'),
        ('ediscf_x12_204_commodity_pieces', 'repeat.pieces', 'repeat', 'integer', TRUE, 'commodities', 'repeat', 'Commodity Pieces', 'Commodity pieces.', 'Active'),
        ('ediscf_x12_204_commodity_sequence', 'repeat.sequence', 'repeat', 'integer', TRUE, 'commodities', 'repeat', 'Commodity Sequence', 'Optional commodity sequence used by templates.', 'Active'),

        ('ediscf_x12_204_charge_id', 'repeat.accessorialChargeId', 'repeat', 'string', TRUE, 'additionalCharges', 'repeat', 'Accessorial Charge ID', 'Accessorial charge identifier.', 'Active'),
        ('ediscf_x12_204_charge_label', 'repeat.accessorialLabel', 'repeat', 'string', TRUE, 'additionalCharges', 'repeat', 'Accessorial Label', 'Accessorial display label.', 'Active'),
        ('ediscf_x12_204_charge_code', 'repeat.accessorialCode', 'repeat', 'string', TRUE, 'additionalCharges', 'repeat', 'Accessorial Code', 'Accessorial charge code.', 'Active'),
        ('ediscf_x12_204_charge_description', 'repeat.accessorialDescription', 'repeat', 'string', TRUE, 'additionalCharges', 'repeat', 'Accessorial Description', 'Accessorial description.', 'Active'),
        ('ediscf_x12_204_charge_method', 'repeat.method', 'repeat', 'string', TRUE, 'additionalCharges', 'repeat', 'Accessorial Method', 'Accessorial charge method.', 'Active'),
        ('ediscf_x12_204_charge_amount', 'repeat.amount', 'repeat', 'decimal', TRUE, 'additionalCharges', 'repeat', 'Accessorial Amount', 'Accessorial charge amount.', 'Active'),
        ('ediscf_x12_204_charge_unit', 'repeat.unit', 'repeat', 'integer', TRUE, 'additionalCharges', 'repeat', 'Accessorial Unit', 'Accessorial charge unit.', 'Active'),

        ('ediscf_x12_204_partner_carrier_scac', 'partner.carrier.scac', 'partner', 'string', FALSE, NULL, 'partner.carrier', 'Carrier SCAC', 'Partner carrier SCAC.', 'Active'),
        ('ediscf_x12_204_partner_contact_name', 'partner.contact.name', 'partner', 'string', FALSE, NULL, 'partner.contact', 'Contact Name', 'Partner contact name.', 'Active'),
        ('ediscf_x12_204_partner_contact_email', 'partner.contact.email', 'partner', 'string', FALSE, NULL, 'partner.contact', 'Contact Email', 'Partner contact email.', 'Active'),
        ('ediscf_x12_204_partner_contact_phone', 'partner.contact.phone', 'partner', 'string', FALSE, NULL, 'partner.contact', 'Contact Phone', 'Partner contact phone.', 'Active'),
        ('ediscf_x12_204_partner_shipper_code', 'partner.shipper.code', 'partner', 'string', FALSE, NULL, 'partner.shipper', 'Shipper Code', 'Partner shipper code.', 'Active'),
        ('ediscf_x12_204_partner_shipper_name', 'partner.shipper.name', 'partner', 'string', FALSE, NULL, 'partner.shipper', 'Shipper Name', 'Partner shipper name.', 'Active'),
        ('ediscf_x12_204_partner_consignee_code', 'partner.consignee.code', 'partner', 'string', FALSE, NULL, 'partner.consignee', 'Consignee Code', 'Partner consignee code.', 'Active'),
        ('ediscf_x12_204_partner_consignee_name', 'partner.consignee.name', 'partner', 'string', FALSE, NULL, 'partner.consignee', 'Consignee Name', 'Partner consignee name.', 'Active'),
        ('ediscf_x12_204_partner_equipment_type', 'partner.equipment.type', 'partner', 'string', FALSE, NULL, 'partner.equipment', 'Equipment Type', 'Partner equipment type.', 'Active'),
        ('ediscf_x12_204_partner_reference_bol_qualifier', 'partner.reference.bolQualifier', 'partner', 'string', FALSE, NULL, 'partner.reference', 'BOL Qualifier', 'Partner BOL reference qualifier.', 'Active'),

        ('ediscf_x12_204_runtime_interchange_sender', 'runtime.interchangeSenderId', 'runtime', 'string', FALSE, NULL, 'runtime', 'Interchange Sender ID', 'Padded ISA sender ID.', 'Active'),
        ('ediscf_x12_204_runtime_interchange_receiver', 'runtime.interchangeReceiverId', 'runtime', 'string', FALSE, NULL, 'runtime', 'Interchange Receiver ID', 'Padded ISA receiver ID.', 'Active'),
        ('ediscf_x12_204_runtime_application_sender', 'runtime.applicationSenderCode', 'runtime', 'string', FALSE, NULL, 'runtime', 'Application Sender Code', 'GS sender code.', 'Active'),
        ('ediscf_x12_204_runtime_application_receiver', 'runtime.applicationReceiverCode', 'runtime', 'string', FALSE, NULL, 'runtime', 'Application Receiver Code', 'GS receiver code.', 'Active'),
        ('ediscf_x12_204_runtime_usage_indicator', 'runtime.usageIndicator', 'runtime', 'string', FALSE, NULL, 'runtime', 'Usage Indicator', 'ISA usage indicator.', 'Active'),
        ('ediscf_x12_204_runtime_component_separator', 'runtime.componentSeparator', 'runtime', 'string', FALSE, NULL, 'runtime', 'Component Separator', 'X12 component separator.', 'Active'),
        ('ediscf_x12_204_runtime_repetition_separator', 'runtime.repetitionSeparator', 'runtime', 'string', FALSE, NULL, 'runtime', 'Repetition Separator', 'X12 repetition separator.', 'Active'),
        ('ediscf_x12_204_runtime_functional_group', 'runtime.functionalGroupId', 'runtime', 'string', FALSE, NULL, 'runtime', 'Functional Group ID', 'Functional group ID.', 'Active'),
        ('ediscf_x12_204_runtime_x12_version', 'runtime.x12Version', 'runtime', 'string', FALSE, NULL, 'runtime', 'X12 Version', 'Runtime X12 version.', 'Active'),
        ('ediscf_x12_204_runtime_interchange_date', 'runtime.interchangeDate', 'runtime', 'date', FALSE, NULL, 'runtime', 'Interchange Date', 'ISA date.', 'Active'),
        ('ediscf_x12_204_runtime_interchange_time', 'runtime.interchangeTime', 'runtime', 'string', FALSE, NULL, 'runtime', 'Interchange Time', 'ISA time.', 'Active'),
        ('ediscf_x12_204_runtime_group_date', 'runtime.groupDate', 'runtime', 'date', FALSE, NULL, 'runtime', 'Group Date', 'GS date.', 'Active'),
        ('ediscf_x12_204_runtime_group_time', 'runtime.groupTime', 'runtime', 'string', FALSE, NULL, 'runtime', 'Group Time', 'GS time.', 'Active'),
        ('ediscf_x12_204_runtime_isa_control', 'runtime.isaControlNumber', 'runtime', 'string', FALSE, NULL, 'runtime', 'ISA Control Number', 'Interchange control number.', 'Active'),
        ('ediscf_x12_204_runtime_group_control', 'runtime.groupControlNumber', 'runtime', 'string', FALSE, NULL, 'runtime', 'Group Control Number', 'Functional group control number.', 'Active'),
        ('ediscf_x12_204_runtime_transaction_control', 'runtime.transactionControlNumber', 'runtime', 'string', FALSE, NULL, 'runtime', 'Transaction Control Number', 'Transaction control number.', 'Active'),
        ('ediscf_x12_204_runtime_transaction_segment_count', 'runtime.transactionSegmentCount', 'runtime', 'integer', FALSE, NULL, 'runtime', 'Transaction Segment Count', 'Rendered ST through SE segment count.', 'Active'),

        ('ediscf_x12_204_mapping_customer', 'mapping.customer', 'mapping', 'object', FALSE, NULL, 'mapping', 'Customer Mapping', 'Resolved customer mapping context.', 'Future'),
        ('ediscf_x12_204_mapping_service_type', 'mapping.serviceType', 'mapping', 'object', FALSE, NULL, 'mapping', 'Service Type Mapping', 'Resolved service type mapping context.', 'Future'),
        ('ediscf_x12_204_mapping_shipment_type', 'mapping.shipmentType', 'mapping', 'object', FALSE, NULL, 'mapping', 'Shipment Type Mapping', 'Resolved shipment type mapping context.', 'Future'),
        ('ediscf_x12_204_mapping_location', 'mapping.location', 'mapping', 'object', FALSE, NULL, 'mapping', 'Location Mapping', 'Resolved location mapping context.', 'Future'),
        ('ediscf_x12_204_mapping_commodity', 'mapping.commodity', 'mapping', 'object', FALSE, NULL, 'mapping', 'Commodity Mapping', 'Resolved commodity mapping context.', 'Future'),
        ('ediscf_x12_204_mapping_accessorial', 'mapping.accessorialCharge', 'mapping', 'object', FALSE, NULL, 'mapping', 'Accessorial Charge Mapping', 'Resolved accessorial charge mapping context.', 'Future')
)
INSERT INTO "edi_source_context_fields"(
    "id",
    "schema_id",
    "path",
    "source_kind",
    "data_type",
    "repeated",
    "repeat_path",
    "parent_path",
    "display_name",
    "description",
    "status"
)
SELECT
    fields."id",
    schema_row."id",
    fields."path",
    fields."source_kind"::edi_source_context_kind_enum,
    fields."data_type"::edi_source_context_data_type_enum,
    fields."repeated",
    fields."repeat_path",
    fields."parent_path",
    fields."display_name",
    fields."description",
    fields."status"::edi_source_context_field_status_enum
FROM fields
CROSS JOIN schema_row
ON CONFLICT DO NOTHING;
