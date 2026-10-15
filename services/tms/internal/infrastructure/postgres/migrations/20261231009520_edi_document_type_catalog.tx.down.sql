DELETE FROM "edi_document_types" AS edt
WHERE edt."id" IN (
        'edidt_x12_204_inbound',
        'edidt_x12_990_outbound',
        'edidt_x12_214_outbound',
        'edidt_x12_210_outbound',
        'edidt_x12_997_outbound',
        'edidt_x12_999_outbound'
    )
    AND NOT EXISTS (SELECT 1 FROM "edi_templates" AS et WHERE et."document_type_id" = edt."id")
    AND NOT EXISTS (SELECT 1 FROM "edi_partner_document_profiles" AS p WHERE p."document_type_id" = edt."id")
    AND NOT EXISTS (SELECT 1 FROM "edi_messages" AS m WHERE m."document_type_id" = edt."id")
    AND NOT EXISTS (SELECT 1 FROM "edi_control_number_sequences" AS s WHERE s."document_type_id" = edt."id");
