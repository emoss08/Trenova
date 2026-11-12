-- A share can be taken back. The row is kept, marked with who took it back
-- and when, so who could see a record through a share stays a matter of
-- record; a revoked share lets its recipient see nothing. Sharing the record
-- with the same person again clears the mark.
ALTER TABLE "record_shares"
    ADD COLUMN IF NOT EXISTS "revoked_at" bigint,
    ADD COLUMN IF NOT EXISTS "revoked_by_id" varchar(100);

ALTER TABLE "record_shares"
    DROP CONSTRAINT IF EXISTS "fk_record_shares_revoked_by",
    ADD CONSTRAINT "fk_record_shares_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

ALTER TABLE "record_shares"
    DROP CONSTRAINT IF EXISTS "ck_record_shares_revoked",
    ADD CONSTRAINT "ck_record_shares_revoked" CHECK ("revoked_at" IS NOT NULL OR "revoked_by_id" IS NULL);

COMMENT ON TABLE "record_shares" IS 'Who shared which record with whom: one row per record and recipient, counted and re-dated each time it is shared again, and kept when the share is taken back. A share lets its recipient read the record''s case; it grants nothing else';

COMMENT ON COLUMN "record_shares"."revoked_at" IS 'When the share was taken back; null while it stands';

COMMENT ON COLUMN "record_shares"."revoked_by_id" IS 'Who took the share back';
