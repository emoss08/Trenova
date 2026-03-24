DELETE FROM "assignments"
WHERE "primary_worker_id" IS NULL
   OR "tractor_id" IS NULL;

ALTER TABLE "assignments"
    ALTER COLUMN "primary_worker_id" SET NOT NULL,
    ALTER COLUMN "tractor_id" SET NOT NULL;
