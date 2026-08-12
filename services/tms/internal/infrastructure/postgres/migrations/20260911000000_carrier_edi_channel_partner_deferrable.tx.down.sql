ALTER TABLE "carrier_edi_channels"
    DROP CONSTRAINT IF EXISTS "uq_carrier_edi_channels_partner";

--bun:split

CREATE UNIQUE INDEX "uq_carrier_edi_channels_partner" ON "carrier_edi_channels"("carrier_id", "edi_partner_id", "organization_id", "business_unit_id");
