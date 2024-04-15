-- Modify "shipment_charges" table
ALTER TABLE "shipment_charges" ADD COLUMN "shipment_id" uuid NOT NULL, ADD COLUMN "accessorial_charge_id" uuid NOT NULL, ADD COLUMN "description" text NOT NULL, ADD COLUMN "charge_amount" numeric(19,4) NOT NULL, ADD COLUMN "units" bigint NOT NULL, ADD COLUMN "sub_total" numeric(19,4) NOT NULL, ADD COLUMN "created_by" uuid NULL;
-- Modify "shipments" table
ALTER TABLE "shipments" ALTER COLUMN "other_charge_amount" TYPE numeric(19,4), ALTER COLUMN "freight_charge_amount" TYPE numeric(19,4), ALTER COLUMN "pieces" TYPE numeric(10,2), ALTER COLUMN "weight" TYPE numeric(10,2), ALTER COLUMN "total_charge_amount" TYPE numeric(19,4), ALTER COLUMN "current_suffix" TYPE character varying(2);
