UPDATE "shipment_events"
SET "metadata" = "metadata" - 'marginPercent'
WHERE "type" = 'TenderCountered'
  AND "metadata" ? 'marginPercent';
