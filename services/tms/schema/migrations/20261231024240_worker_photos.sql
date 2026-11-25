-- A worker's photo is two stored objects, the normalized picture and its thumbnail, kept as
-- storage keys the server signs on read. The old column held whatever a client put in an
-- <img src>, which was a key, not a URL.
ALTER TABLE "workers"
    DROP COLUMN IF EXISTS "profile_pic_url",
    ADD COLUMN IF NOT EXISTS "photo_key" varchar(255),
    ADD COLUMN IF NOT EXISTS "photo_thumbnail_key" varchar(255),
    ADD CONSTRAINT "ck_workers_photo_pair" CHECK (("photo_key" IS NULL) = ("photo_thumbnail_key" IS NULL));

-- A driver's proposed photo waits on the office as staged objects. A photo request and a
-- contact request are decided separately, so each kind may have one request waiting.
ALTER TABLE "worker_profile_change_requests"
    ADD COLUMN IF NOT EXISTS "proposed_photo_key" varchar(255),
    ADD COLUMN IF NOT EXISTS "proposed_photo_thumbnail_key" varchar(255),
    ADD CONSTRAINT "ck_worker_profile_change_requests_photo_pair" CHECK (("proposed_photo_key" IS NULL) = ("proposed_photo_thumbnail_key" IS NULL));

DROP INDEX IF EXISTS "uq_worker_profile_change_requests_open";

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_profile_change_requests_open" ON "worker_profile_change_requests"("organization_id", "business_unit_id", "worker_id")
WHERE
    "status" = 'Pending' AND "proposed_photo_key" IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_profile_change_requests_open_photo" ON "worker_profile_change_requests"("organization_id", "business_unit_id", "worker_id")
WHERE
    "status" = 'Pending' AND "proposed_photo_key" IS NOT NULL;
