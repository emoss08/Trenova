ALTER TABLE "shipment_holds"
    DROP CONSTRAINT IF EXISTS "fk_shipment_holds_hold_reason";

DROP INDEX IF EXISTS "idx_shipment_holds_hold_reason";

ALTER TABLE "shipment_holds"
    DROP COLUMN IF EXISTS "hold_reason_id";

CREATE OR REPLACE FUNCTION protect_shipment_holds()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF OLD.released_at IS NOT NULL THEN
        RAISE EXCEPTION 'Cannot modify released holds';
    END IF;
    IF NEW.id <> OLD.id OR NEW.shipment_id <> OLD.shipment_id OR NEW.organization_id <> OLD.organization_id OR NEW.business_unit_id <> OLD.business_unit_id THEN
        RAISE EXCEPTION 'Scope fields are immutable';
    END IF;
    IF NEW.type <> OLD.type THEN
        RAISE EXCEPTION 'Hold type is immutable; release and create a new hold to reclassify';
    END IF;
    IF NEW.source <> OLD.source THEN
        RAISE EXCEPTION 'Source is immutable';
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;
