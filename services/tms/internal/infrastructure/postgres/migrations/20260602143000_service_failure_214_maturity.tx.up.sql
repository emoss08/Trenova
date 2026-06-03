ALTER TYPE edi_mapping_entity_type_enum ADD VALUE IF NOT EXISTS 'ServiceFailureReasonCode';

--bun:split
CREATE TYPE edi_message_ack_status_enum AS ENUM(
    'NotExpected',
    'Pending',
    'Accepted',
    'Rejected',
    'Failed'
);

ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "ack_status" edi_message_ack_status_enum,
    ADD COLUMN IF NOT EXISTS "ack_message_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "ack_received_at" bigint,
    ADD COLUMN IF NOT EXISTS "ack_last_error" text;

CREATE INDEX IF NOT EXISTS "idx_edi_messages_ack"
    ON "edi_messages"("organization_id", "business_unit_id", "ack_status", "generated_at" DESC)
    WHERE "ack_status" IS NOT NULL;

--bun:split
INSERT INTO "edi_source_context_fields"(
    "id",
    "schema_id",
    "path",
    "source_kind",
    "data_type",
    "display_name",
    "description",
    "status"
)
SELECT
    'ediscf_x12_214_event_time_code',
    'edisc_x12_214_out_shipment_status_v1',
    'shipmentStatus.eventTimeCode',
    'shipment',
    'string',
    'Event Time Code',
    'X12 AT7 event time code.',
    'Active'
WHERE EXISTS (
    SELECT 1 FROM "edi_source_context_schemas"
    WHERE "id" = 'edisc_x12_214_out_shipment_status_v1'
)
ON CONFLICT DO NOTHING;
