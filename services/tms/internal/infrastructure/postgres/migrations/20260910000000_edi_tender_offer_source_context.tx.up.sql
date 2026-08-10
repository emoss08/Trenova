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
    'ediscf_x12_204_tender_offer_id',
    schema_row."id",
    'shipment.tenderOfferId',
    'shipment'::edi_source_context_kind_enum,
    'string'::edi_source_context_data_type_enum,
    FALSE,
    NULL,
    'shipment',
    'Tender Offer ID',
    'Carrier tender offer identifier echoed back on the 990 response.',
    'Active'::edi_source_context_field_status_enum
FROM (
    SELECT "id"
    FROM "edi_source_context_schemas"
    WHERE "id" = 'edisc_x12_204_out_load_tender_v1'
) AS schema_row
ON CONFLICT DO NOTHING;
