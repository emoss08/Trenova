ALTER TABLE "shipment_holds"
    ADD COLUMN IF NOT EXISTS "hold_reason_id" varchar(100);

ALTER TABLE "shipment_holds"
    ADD CONSTRAINT "fk_shipment_holds_hold_reason"
        FOREIGN KEY ("hold_reason_id", "organization_id")
        REFERENCES "hold_reasons"("id", "organization_id")
        ON UPDATE NO ACTION
        ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_hold_reason" ON "shipment_holds"("hold_reason_id");

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
    IF COALESCE(NEW.hold_reason_id, '') <> COALESCE(OLD.hold_reason_id, '') THEN
        RAISE EXCEPTION 'Hold reason is immutable; release and create a new hold to change the reason';
    END IF;
    IF COALESCE(NEW.reason_code, '') <> COALESCE(OLD.reason_code, '') THEN
        RAISE EXCEPTION 'Reason code is immutable';
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;
