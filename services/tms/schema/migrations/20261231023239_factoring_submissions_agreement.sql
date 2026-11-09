--keel:notx
-- The agreement page reads what the schedules sent under one agreement carried, index-only.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_submissions_agreement" ON "factoring_submissions"("organization_id", "business_unit_id", "agreement_id") INCLUDE ("status", "sent_at", "invoice_count", "face_minor", "expected_advance_minor", "expected_fee_minor");
