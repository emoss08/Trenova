--keel:notx
-- The report catalog check finds saved reports last checked against another catalog version,
-- across every organization, with two range scans that are empty once a deploy has been checked.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_report_definitions_catalog_version" ON "report_definitions"("catalog_version");
