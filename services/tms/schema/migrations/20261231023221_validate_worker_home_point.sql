-- Validates the home point check added NOT VALID, so adding it never held the workers lock
-- while scanning the table.
ALTER TABLE "workers" VALIDATE CONSTRAINT "ck_workers_home_point";
