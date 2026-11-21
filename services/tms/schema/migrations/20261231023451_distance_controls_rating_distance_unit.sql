-- The unit tariffs and rate tables read distance in was named for stored mileage, which it never
-- governed. It is the rating unit.
ALTER TABLE "distance_controls" RENAME COLUMN "stored_distance_units" TO "rating_distance_unit";

ALTER TABLE "distance_controls" RENAME CONSTRAINT "ck_distance_controls_stored_units"
    TO "ck_distance_controls_rating_distance_unit";
