ALTER TABLE "factoring_remittances" ADD COLUMN IF NOT EXISTS "sender_email" varchar(320);

ALTER TABLE "factoring_remittance_lines" DROP CONSTRAINT IF EXISTS "chk_factoring_remittance_lines_applied";

SELECT trenova_rls.reconcile();
