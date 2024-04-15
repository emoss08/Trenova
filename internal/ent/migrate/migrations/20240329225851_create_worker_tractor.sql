-- Modify "tractors" table
ALTER TABLE "tractors" DROP COLUMN "primary_worker_id", DROP COLUMN "secondary_worker_id";
-- Modify "workers" table
ALTER TABLE "workers" ADD COLUMN "primary_worker_id" uuid NULL, ADD CONSTRAINT "workers_tractors_primary_worker" FOREIGN KEY ("primary_worker_id") REFERENCES "tractors" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "workers_primary_worker_id_key" to table: "workers"
CREATE UNIQUE INDEX "workers_primary_worker_id_key" ON "workers" ("primary_worker_id");
