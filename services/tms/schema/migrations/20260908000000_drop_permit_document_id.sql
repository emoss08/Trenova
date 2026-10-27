-- permits.document_id was declared and never used: nothing in the application
-- ever read or wrote it. Permit paperwork attaches through the general document
-- system instead, keyed on (resource_type, resource_id) like every other
-- document, which is what gives it the same versioning, retention and access
-- rules rather than a second parallel mechanism.
ALTER TABLE "permits"
    DROP COLUMN IF EXISTS "document_id";
