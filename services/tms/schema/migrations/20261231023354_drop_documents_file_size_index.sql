--keel:notx
-- The storage level is read from document_storage_totals now, so nothing adds up every
-- document's size and the index that served it goes.
DROP INDEX CONCURRENTLY IF EXISTS "idx_documents_file_size";
