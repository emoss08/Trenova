ALTER TABLE "factoring_remittance_lines" ADD CONSTRAINT "chk_factoring_remittance_lines_applied" CHECK ("match_status" <> 'Applied' OR "kind" = 'Rejection' OR "reserve_entry_id" IS NOT NULL) NOT VALID;

ALTER TABLE "factoring_remittances" DROP COLUMN IF EXISTS "sender_email";
