DROP INDEX IF EXISTS idx_invoice_adjustment_document_references_adjustment;

--bun:split
DROP TABLE IF EXISTS invoice_adjustment_document_references;
