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
    LEFT(s."schema_id" || '_rt_isa_sender_qual', 100),
    s."schema_id",
    'runtime.interchangeSenderQualifier',
    'runtime'::edi_source_context_kind_enum,
    'string'::edi_source_context_data_type_enum,
    FALSE,
    NULL,
    'runtime',
    'Interchange Sender Qualifier',
    'ISA05 interchange sender ID qualifier from the document profile envelope.',
    'Active'::edi_source_context_field_status_enum
FROM (
    SELECT DISTINCT "schema_id"
    FROM "edi_source_context_fields"
    WHERE "path" = 'runtime.interchangeSenderId'
) s
ON CONFLICT DO NOTHING;

--bun:split
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
    LEFT(s."schema_id" || '_rt_isa_receiver_qual', 100),
    s."schema_id",
    'runtime.interchangeReceiverQualifier',
    'runtime'::edi_source_context_kind_enum,
    'string'::edi_source_context_data_type_enum,
    FALSE,
    NULL,
    'runtime',
    'Interchange Receiver Qualifier',
    'ISA07 interchange receiver ID qualifier from the document profile envelope.',
    'Active'::edi_source_context_field_status_enum
FROM (
    SELECT DISTINCT "schema_id"
    FROM "edi_source_context_fields"
    WHERE "path" = 'runtime.interchangeReceiverId'
) s
ON CONFLICT DO NOTHING;
