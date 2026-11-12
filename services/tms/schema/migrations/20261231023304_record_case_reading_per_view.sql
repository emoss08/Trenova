-- A reading is written from the signals its reader may see, and two people
-- with different sight of one record have different signals. Kept once per
-- record, each of them replaced the other's reading on every look and it was
-- written again without end. A record now keeps one reading for each set of
-- signals it was read from, so neither replaces the other and nobody is shown
-- a reading drawn from signals they may not see.
DROP INDEX IF EXISTS "uq_record_case_readings_record";

CREATE UNIQUE INDEX IF NOT EXISTS "uq_record_case_readings_record_fingerprint" ON "record_case_readings"("organization_id", "business_unit_id", "record_id", "fingerprint");

COMMENT ON TABLE "record_case_readings" IS 'Nova''s reading of a record''s case: the verdict with its citations and highlights, kept per record and set of open signals it was read from';
