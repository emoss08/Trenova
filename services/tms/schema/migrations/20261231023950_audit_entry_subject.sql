--keel:lint_ignore index-concurrently,lock-heavy
-- An audit entry names the record whose file it belongs to: the worker for a change to one of the
-- worker's credentials, or the record itself for a root record, so a worker's whole history is one
-- index range. The writer puts the subject in metadata, which the sealed digest covers, and these
-- columns are its projection; the digest lists its columns, so they leave seals and verification
-- unchanged. Nullable columns without a default are a catalog change on every partition, and
-- ensure_partitions creates new partitions LIKE the parent, so they carry them. The table is
-- partitioned, which rules out building the index concurrently; no existing entry has a subject,
-- so the partial index and the check read the partitions once and keep nothing.
ALTER TABLE "audit_entries"
    ADD COLUMN IF NOT EXISTS "subject_resource" VARCHAR(50),
    ADD COLUMN IF NOT EXISTS "subject_id" VARCHAR(100);

ALTER TABLE "audit_entries"
    ADD CONSTRAINT "ck_audit_entries_subject" CHECK (("subject_resource" IS NULL) = ("subject_id" IS NULL));

COMMENT ON COLUMN "audit_entries"."subject_resource" IS 'The kind of the record whose file the entry belongs to (the worker for a change to a worker''s credential, the record itself for a root record); a projection of metadata.subjectResource, which the seal covers';

COMMENT ON COLUMN "audit_entries"."subject_id" IS 'The id of the record whose file the entry belongs to; a projection of metadata.subjectId, which the seal covers. Null on entries written before subjects were recorded';

CREATE INDEX IF NOT EXISTS "idx_audit_entries_tenant_subject" ON "audit_entries"("organization_id", "business_unit_id", "subject_id", "timestamp" DESC, "id" DESC) WHERE "subject_id" IS NOT NULL;
