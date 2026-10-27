ALTER TABLE "report_runs"
    ADD COLUMN IF NOT EXISTS "sample" BYTEA;

COMMENT ON COLUMN "report_runs"."sample" IS
    'The first rows and the grand total of a finished run: the keel export JSON envelope its digest captured, at most 48 KB, kept as bytes because the envelope reader needs its keys in written order. Lets the result be read where it is shown without opening the rows sidecar. Cleared when the run expires.';
