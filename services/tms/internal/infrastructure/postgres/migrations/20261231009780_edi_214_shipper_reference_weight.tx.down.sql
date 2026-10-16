DELETE FROM "edi_source_context_fields"
WHERE "path" IN ('shipmentStatus.shipperReference', 'shipmentStatus.weight', 'shipmentStatus.pieces');
