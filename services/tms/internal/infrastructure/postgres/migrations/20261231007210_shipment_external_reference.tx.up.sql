--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The customer's own identifier for a load (an EDI 204's shipment identification
-- number, an order number sent through the API). One customer's reference names one
-- live shipment, so a resent tender or a retried request cannot book the load twice;
-- a canceled shipment releases its reference.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "external_reference" varchar(100);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipments_external_reference" ON "shipments"("organization_id", "business_unit_id", "customer_id", lower("external_reference"))
WHERE
    "external_reference" IS NOT NULL
    AND "status" <> 'Canceled';
