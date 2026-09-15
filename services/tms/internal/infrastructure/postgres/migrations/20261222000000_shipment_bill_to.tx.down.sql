DROP INDEX IF EXISTS "idx_shipments_bill_to_customer";

--bun:split
ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "fk_shipments_bill_to_customer";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "bill_to_customer_id",
    DROP COLUMN IF EXISTS "freight_terms";

--bun:split
DROP TYPE IF EXISTS "shipment_freight_terms_enum";
