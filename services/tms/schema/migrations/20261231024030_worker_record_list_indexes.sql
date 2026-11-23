--keel:notx
-- The worker record page pages each of a worker's lists newest first with a keyset cursor. Each
-- index leads with the tenant and the worker and carries the list's whole order, so a page is one
-- range scan that stops at the page size, whatever the length of the worker's history.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_safety_events_worker_page" ON "worker_safety_events"("organization_id", "business_unit_id", "worker_id", "occurred_at" DESC, "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_training_records_worker_page" ON "worker_training_records"("organization_id", "business_unit_id", "worker_id", "assigned_at" DESC, "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_credentials_worker_page" ON "worker_credentials"("organization_id", "business_unit_id", "worker_id", "status", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_disciplinary_actions_worker_page" ON "worker_disciplinary_actions"("organization_id", "business_unit_id", "worker_id", "issued_at" DESC, "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_recognitions_worker_page" ON "worker_recognitions"("organization_id", "business_unit_id", "worker_id", "occurred_at" DESC, "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_injuries_worker_page" ON "worker_injuries"("organization_id", "business_unit_id", "worker_id", "case_year" DESC, "case_number" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_benefit_enrollments_worker_page" ON "worker_benefit_enrollments"("organization_id", "business_unit_id", "worker_id", "effective_from" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_training_records_worker_course" ON "worker_training_records"("organization_id", "business_unit_id", "worker_id", "course_id");
