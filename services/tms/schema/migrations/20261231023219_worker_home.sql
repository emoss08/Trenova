-- Where a driver lives and when they were last there, so the planner can price time away
-- from home and credit a move that heads home. The coordinates come from the address the
-- worker form looked up; last_home_at is recorded by a dispatcher or stamped by the
-- telematics feed when the driver's tractor is seen near home.
ALTER TABLE "workers"
    ADD COLUMN IF NOT EXISTS "home_latitude" double precision,
    ADD COLUMN IF NOT EXISTS "home_longitude" double precision,
    ADD COLUMN IF NOT EXISTS "home_place_id" varchar(255),
    ADD COLUMN IF NOT EXISTS "last_home_at" bigint;

ALTER TABLE "workers"
    ADD CONSTRAINT "ck_workers_home_point" CHECK (
        ("home_latitude" IS NULL) = ("home_longitude" IS NULL)
        AND ("home_latitude" IS NULL OR "home_latitude" BETWEEN -90 AND 90)
        AND ("home_longitude" IS NULL OR "home_longitude" BETWEEN -180 AND 180)
    ) NOT VALID;
