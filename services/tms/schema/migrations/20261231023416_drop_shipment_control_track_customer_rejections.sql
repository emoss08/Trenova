-- Tracking customer rejections was a stored switch that nothing read: no record of a customer
-- refusing a shipment exists for it to gate.
ALTER TABLE "shipment_controls"
    DROP COLUMN IF EXISTS "track_customer_rejections";
