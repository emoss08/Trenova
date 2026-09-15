-- A shipment may be paid by a customer other than the one who ordered it. The
-- bill-to is optional: absent means the ordering customer pays, which is what
-- every existing shipment does today.
DO $$
BEGIN
    CREATE TYPE "shipment_freight_terms_enum" AS ENUM (
        'Prepaid',
        'Collect',
        'ThirdParty'
    );
EXCEPTION
    WHEN duplicate_object THEN
        NULL;
END
$$;

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "bill_to_customer_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "freight_terms" shipment_freight_terms_enum NOT NULL DEFAULT 'Prepaid';

--bun:split
ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_bill_to_customer" FOREIGN KEY ("bill_to_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_bill_to_customer" ON "shipments"("bill_to_customer_id", "organization_id", "business_unit_id")
WHERE
    "bill_to_customer_id" IS NOT NULL;
