DELETE FROM "edi_source_context_fields"
WHERE "path" IN ('runtime.interchangeSenderQualifier', 'runtime.interchangeReceiverQualifier');
