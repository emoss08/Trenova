-- A shipment stores when its status last changed, so "loads sitting in New for three days" is a
-- column comparison rather than a read of its history. The database keeps it: the application never
-- writes the column. An inserted shipment takes its created_at, and an update that changes the
-- status takes the time of that write. Existing rows take their created_at when still New, which is
-- exact, and their updated_at otherwise, the latest moment the status could have changed, so an age
-- read from a backfilled row is never more than the truth.
ALTER TABLE "shipments" ADD COLUMN IF NOT EXISTS "status_changed_at" BIGINT;

UPDATE "shipments"
SET "status_changed_at" = CASE
    WHEN "status" = 'New' THEN "created_at"
    ELSE greatest("created_at", "updated_at")
END
WHERE "status_changed_at" IS NULL;

ALTER TABLE "shipments" ALTER COLUMN "status_changed_at" SET NOT NULL;

CREATE OR REPLACE FUNCTION trenova_shipment_status_changed_at()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW."status_changed_at" := coalesce(nullif(NEW."created_at", 0), extract(epoch FROM now())::bigint);
    ELSIF NEW."status" IS DISTINCT FROM OLD."status" THEN
        NEW."status_changed_at" := extract(epoch FROM now())::bigint;
    ELSE
        NEW."status_changed_at" := OLD."status_changed_at";
    END IF;
    RETURN NEW;
END
$$;

CREATE OR REPLACE TRIGGER "trg_shipments_status_changed_at"
    BEFORE INSERT OR UPDATE OF "status", "status_changed_at" ON "shipments"
    FOR EACH ROW
    EXECUTE FUNCTION trenova_shipment_status_changed_at();
