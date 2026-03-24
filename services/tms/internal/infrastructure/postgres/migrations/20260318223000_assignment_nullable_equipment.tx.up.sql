ALTER TABLE "assignments"
    ALTER COLUMN "primary_worker_id" DROP NOT NULL,
    ALTER COLUMN "tractor_id" DROP NOT NULL;
