UPDATE "factoring_invoice_states" SET "verdict" = 'NotReady', "ready_since" = NULL WHERE "verdict" = 'Submitted';
ALTER TABLE "factoring_invoice_states" DROP CONSTRAINT IF EXISTS "chk_factoring_invoice_states_verdict";
ALTER TABLE "factoring_invoice_states" ADD CONSTRAINT "chk_factoring_invoice_states_verdict" CHECK ("verdict" IN ('Ready', 'NotReady', 'Excluded'));
DROP TABLE IF EXISTS "factoring_submission_items";
DROP TABLE IF EXISTS "factoring_submissions";
DROP TABLE IF EXISTS "factoring_transports";
