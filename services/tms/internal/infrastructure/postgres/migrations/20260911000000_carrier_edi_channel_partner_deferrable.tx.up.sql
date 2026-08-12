DROP INDEX IF EXISTS "uq_carrier_edi_channels_partner";

--bun:split

ALTER TABLE "carrier_edi_channels"
    ADD CONSTRAINT "uq_carrier_edi_channels_partner"
    UNIQUE ("carrier_id", "edi_partner_id", "organization_id", "business_unit_id")
    DEFERRABLE INITIALLY IMMEDIATE;
