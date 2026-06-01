WITH service_failure_resources(resource, operations) AS (
    VALUES
        ('service_failure', ARRAY['read', 'create', 'update', 'approve', 'archive', 'export']::text[]),
        ('service_failure_reason_code', ARRAY['read', 'create', 'update', 'approve', 'archive', 'export']::text[])
)
INSERT INTO resource_permissions(
    id,
    role_id,
    resource,
    operations,
    data_scope,
    created_at,
    updated_at
)
SELECT
    CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
    r.id,
    sfr.resource,
    sfr.operations,
    'organization',
    EXTRACT(EPOCH FROM current_timestamp)::bigint,
    EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM roles r
CROSS JOIN service_failure_resources sfr
WHERE r.is_system = true
  AND r.name = 'Organization Administrator'
  AND NOT EXISTS (
      SELECT 1
      FROM resource_permissions rp
      WHERE rp.role_id = r.id
        AND rp.resource = sfr.resource
  );

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
    'edisc_x12_214_out_shipment_status_v1',
    NULL,
    NULL,
    'X12',
    '214',
    'Outbound',
    '004010',
    'shipmentStatus',
    1,
    'X12 214 Outbound Shipment Status Source Context',
    'Global source context metadata for outbound X12 214 shipment status templates.',
    'Active'
)
ON CONFLICT DO NOTHING;

WITH schema_row AS (
    SELECT "id" FROM "edi_source_context_schemas"
    WHERE "id" = 'edisc_x12_214_out_shipment_status_v1'
),
fields("id", "path", "source_kind", "data_type", "display_name", "description", "status") AS (
    VALUES
        ('ediscf_x12_214_shipment_id', 'shipmentStatus.shipmentId', 'shipment', 'string', 'Shipment ID', 'Shipment identifier.', 'Active'),
        ('ediscf_x12_214_bol', 'shipmentStatus.bol', 'shipment', 'string', 'Bill of Lading', 'Shipment bill of lading number.', 'Active'),
        ('ediscf_x12_214_pro_number', 'shipmentStatus.proNumber', 'shipment', 'string', 'PRO Number', 'Shipment PRO number.', 'Active'),
        ('ediscf_x12_214_status_code', 'shipmentStatus.statusCode', 'shipment', 'string', 'Shipment Status Code', 'X12 AT7 shipment status code.', 'Active'),
        ('ediscf_x12_214_status_reason_code', 'shipmentStatus.statusReasonCode', 'shipment', 'string', 'Shipment Status Reason Code', 'X12 AT7 shipment status reason code.', 'Active'),
        ('ediscf_x12_214_event_date', 'shipmentStatus.eventDate', 'shipment', 'timestamp', 'Event Date', 'Shipment status event date.', 'Active'),
        ('ediscf_x12_214_event_time', 'shipmentStatus.eventTime', 'shipment', 'timestamp', 'Event Time', 'Shipment status event time.', 'Active'),
        ('ediscf_x12_214_stop_id', 'shipmentStatus.stopId', 'shipment', 'string', 'Stop ID', 'Related stop identifier.', 'Active'),
        ('ediscf_x12_214_stop_type', 'shipmentStatus.stopType', 'shipment', 'string', 'Stop Type', 'Related stop type.', 'Active'),
        ('ediscf_x12_214_stop_sequence', 'shipmentStatus.stopSequence', 'shipment', 'integer', 'Stop Sequence', 'Related stop route sequence.', 'Active'),
        ('ediscf_x12_214_location_id', 'shipmentStatus.locationId', 'shipment', 'string', 'Location ID', 'Related location identifier.', 'Active'),
        ('ediscf_x12_214_location_name', 'shipmentStatus.locationName', 'shipment', 'string', 'Location Name', 'Related location name.', 'Active'),
        ('ediscf_x12_214_location_code', 'shipmentStatus.locationCode', 'shipment', 'string', 'Location Code', 'Related location code.', 'Active'),
        ('ediscf_x12_214_address_line', 'shipmentStatus.addressLine', 'shipment', 'string', 'Address Line', 'Related stop address line.', 'Active'),
        ('ediscf_x12_214_city', 'shipmentStatus.city', 'shipment', 'string', 'City', 'Related event city.', 'Active'),
        ('ediscf_x12_214_state_code', 'shipmentStatus.stateCode', 'shipment', 'string', 'State Code', 'Related event state code.', 'Active'),
        ('ediscf_x12_214_postal_code', 'shipmentStatus.postalCode', 'shipment', 'string', 'Postal Code', 'Related event postal code.', 'Active'),
        ('ediscf_x12_214_country_code', 'shipmentStatus.countryCode', 'shipment', 'string', 'Country Code', 'Related event country code.', 'Active'),
        ('ediscf_x12_214_appointment_number', 'shipmentStatus.appointmentNumber', 'shipment', 'string', 'Appointment Number', 'Appointment or dock schedule reference.', 'Active'),
        ('ediscf_x12_214_scheduled_start', 'shipmentStatus.scheduledWindowStart', 'shipment', 'timestamp', 'Scheduled Window Start', 'Stop scheduled window start.', 'Active'),
        ('ediscf_x12_214_scheduled_end', 'shipmentStatus.scheduledWindowEnd', 'shipment', 'timestamp', 'Scheduled Window End', 'Stop scheduled window end.', 'Active'),
        ('ediscf_x12_214_actual_arrival', 'shipmentStatus.actualArrival', 'shipment', 'timestamp', 'Actual Arrival', 'Stop actual arrival timestamp.', 'Active'),
        ('ediscf_x12_214_actual_departure', 'shipmentStatus.actualDeparture', 'shipment', 'timestamp', 'Actual Departure', 'Stop actual departure timestamp.', 'Active'),
        ('ediscf_x12_214_equipment_number', 'shipmentStatus.equipmentNumber', 'shipment', 'string', 'Equipment Number', 'Trailer, container, or equipment number.', 'Active'),
        ('ediscf_x12_214_equipment_type', 'shipmentStatus.equipmentType', 'shipment', 'string', 'Equipment Type', 'Trailer, container, or equipment type.', 'Active'),
        ('ediscf_x12_214_exception_code', 'shipmentStatus.exceptionCode', 'shipment', 'string', 'Exception Code', 'X12 lading exception code.', 'Active'),
        ('ediscf_x12_214_reason_code', 'shipmentStatus.reasonCode', 'shipment', 'string', 'Reason Code', 'Status or service failure reason code.', 'Active'),
        ('ediscf_x12_214_reason_description', 'shipmentStatus.reasonDescription', 'shipment', 'string', 'Reason Description', 'Status or service failure reason description.', 'Active'),
        ('ediscf_x12_214_late_minutes', 'shipmentStatus.lateMinutes', 'shipment', 'integer', 'Late Minutes', 'Late duration in minutes.', 'Active'),
        ('ediscf_x12_214_service_failure_id', 'shipmentStatus.serviceFailureId', 'shipment', 'string', 'Service Failure ID', 'Future service failure identifier.', 'Active'),
        ('ediscf_x12_214_service_failure_number', 'shipmentStatus.serviceFailureNumber', 'shipment', 'string', 'Service Failure Number', 'Future service failure display number.', 'Active'),
        ('ediscf_x12_214_service_failure_reason_code_id', 'shipmentStatus.serviceFailureReasonCodeId', 'shipment', 'string', 'Service Failure Reason Code ID', 'Future service failure reason code identifier.', 'Active'),
        ('ediscf_x12_214_service_failure_reason_code', 'shipmentStatus.serviceFailureReasonCode', 'shipment', 'string', 'Service Failure Reason Code', 'Future service failure reason code.', 'Active'),
        ('ediscf_x12_214_references', 'shipmentStatus.references', 'shipment', 'object', 'References', 'Shipment status reference values.', 'Active'),
        ('ediscf_x12_214_partner_carrier_scac', 'partner.carrier.scac', 'partner', 'string', 'Carrier SCAC', 'Partner carrier SCAC.', 'Active'),
        ('ediscf_x12_214_runtime_interchange_sender', 'runtime.interchangeSenderId', 'runtime', 'string', 'Interchange Sender ID', 'Padded ISA sender ID.', 'Active'),
        ('ediscf_x12_214_runtime_interchange_receiver', 'runtime.interchangeReceiverId', 'runtime', 'string', 'Interchange Receiver ID', 'Padded ISA receiver ID.', 'Active'),
        ('ediscf_x12_214_runtime_application_sender', 'runtime.applicationSenderCode', 'runtime', 'string', 'Application Sender Code', 'GS sender code.', 'Active'),
        ('ediscf_x12_214_runtime_application_receiver', 'runtime.applicationReceiverCode', 'runtime', 'string', 'Application Receiver Code', 'GS receiver code.', 'Active'),
        ('ediscf_x12_214_runtime_usage_indicator', 'runtime.usageIndicator', 'runtime', 'string', 'Usage Indicator', 'ISA usage indicator.', 'Active'),
        ('ediscf_x12_214_runtime_component_separator', 'runtime.componentSeparator', 'runtime', 'string', 'Component Separator', 'X12 component separator.', 'Active'),
        ('ediscf_x12_214_runtime_repetition_separator', 'runtime.repetitionSeparator', 'runtime', 'string', 'Repetition Separator', 'X12 repetition separator.', 'Active'),
        ('ediscf_x12_214_runtime_functional_group', 'runtime.functionalGroupId', 'runtime', 'string', 'Functional Group ID', 'Functional group ID.', 'Active'),
        ('ediscf_x12_214_runtime_x12_version', 'runtime.x12Version', 'runtime', 'string', 'X12 Version', 'Runtime X12 version.', 'Active'),
        ('ediscf_x12_214_runtime_interchange_date', 'runtime.interchangeDate', 'runtime', 'date', 'Interchange Date', 'ISA date.', 'Active'),
        ('ediscf_x12_214_runtime_interchange_time', 'runtime.interchangeTime', 'runtime', 'string', 'Interchange Time', 'ISA time.', 'Active'),
        ('ediscf_x12_214_runtime_group_date', 'runtime.groupDate', 'runtime', 'date', 'Group Date', 'GS date.', 'Active'),
        ('ediscf_x12_214_runtime_group_time', 'runtime.groupTime', 'runtime', 'string', 'Group Time', 'GS time.', 'Active'),
        ('ediscf_x12_214_runtime_isa_control', 'runtime.isaControlNumber', 'runtime', 'string', 'ISA Control Number', 'Interchange control number.', 'Active'),
        ('ediscf_x12_214_runtime_group_control', 'runtime.groupControlNumber', 'runtime', 'string', 'Group Control Number', 'Functional group control number.', 'Active'),
        ('ediscf_x12_214_runtime_transaction_control', 'runtime.transactionControlNumber', 'runtime', 'string', 'Transaction Control Number', 'Transaction control number.', 'Active'),
        ('ediscf_x12_214_runtime_transaction_segment_count', 'runtime.transactionSegmentCount', 'runtime', 'integer', 'Transaction Segment Count', 'Rendered ST through SE segment count.', 'Active')
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
    FALSE,
    NULL,
    split_part(fields."path", '.', 1),
    fields."display_name",
    fields."description",
    fields."status"::edi_source_context_field_status_enum
FROM fields
CROSS JOIN schema_row
ON CONFLICT DO NOTHING;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_user";

ALTER TABLE "shipment_comments"
    ALTER COLUMN "user_id" DROP NOT NULL,
    ADD CONSTRAINT "ck_shipment_comments_user_source" CHECK ("user_id" IS NOT NULL OR "source" = 'System'),
    ADD CONSTRAINT "fk_shipment_comments_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency",
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_type";

ALTER TABLE "audit_entries"
    ADD CONSTRAINT "chk_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key', 'system')),
    ADD CONSTRAINT "chk_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
        OR
        ("principal_type" = 'system' AND "user_id" IS NULL AND "api_key_id" IS NULL AND "principal_id" IS NOT NULL)
    );
