-- A move is routed with its loaded or empty distance profile. Pay, billing and fuel tax may each
-- measure the same move with a different profile (practical miles for pay, shortest or household
-- goods miles for billing, actual miles per jurisdiction for fuel tax). Each column holds that
-- purpose's distance in the move's distance units, and is empty when the purpose shares the move's
-- profile or the distance was entered by hand, so the move's distance stands for it.
ALTER TABLE "shipment_moves"
    ADD COLUMN "pay_distance" float,
    ADD COLUMN "billing_distance" float,
    ADD COLUMN "fuel_distance" float,
    ADD CONSTRAINT "ck_shipment_moves_pay_distance" CHECK ("pay_distance" >= 0),
    ADD CONSTRAINT "ck_shipment_moves_billing_distance" CHECK ("billing_distance" >= 0),
    ADD CONSTRAINT "ck_shipment_moves_fuel_distance" CHECK ("fuel_distance" >= 0);
