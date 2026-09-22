ALTER TABLE "report_runs"
    ADD COLUMN IF NOT EXISTS "rows_key" VARCHAR(512);

--bun:split

COMMENT ON COLUMN "report_runs"."rows_key" IS
    'Storage key of the JSON rows sidecar written alongside the artifact. NULL marks a run made before the sidecar existed, which is how a comparable run is told from one that cannot be compared.';
