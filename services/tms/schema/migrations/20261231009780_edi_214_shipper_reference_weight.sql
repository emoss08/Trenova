WITH fields("suffix", "path", "data_type", "display_name", "description") AS (
    VALUES
        ('_shipper_ref', 'shipmentStatus.shipperReference', 'string', 'Shipper Reference',
            'The shipper''s own shipment identification: the tender''s shipment ID or the customer reference, else the bill of lading.'),
        ('_weight', 'shipmentStatus.weight', 'integer', 'Weight', 'Shipment gross weight in pounds.'),
        ('_pieces', 'shipmentStatus.pieces', 'integer', 'Pieces', 'Shipment piece count.')
),
schemas AS (
    SELECT DISTINCT "schema_id"
    FROM "edi_source_context_fields"
    WHERE "path" = 'shipmentStatus.proNumber'
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
    LEFT(schemas."schema_id" || fields."suffix", 100),
    schemas."schema_id",
    fields."path",
    'shipment'::edi_source_context_kind_enum,
    fields."data_type"::edi_source_context_data_type_enum,
    FALSE,
    NULL,
    'shipmentStatus',
    fields."display_name",
    fields."description",
    'Active'::edi_source_context_field_status_enum
FROM fields
CROSS JOIN schemas
ON CONFLICT DO NOTHING;
