-- Every report worker records the catalog version it was built with, when it starts and every few
-- minutes after. The catalog check runs only on the newest version still running, so old and new
-- workers of a rolling deploy never undo each other's verdicts, and a rollback hands the check back
-- to the older version once the newer one stops.
CREATE TABLE IF NOT EXISTS "report_catalog_versions"(
    "version" varchar(80) NOT NULL,
    "first_seen_at" bigint NOT NULL,
    "last_seen_at" bigint NOT NULL,
    CONSTRAINT "pk_report_catalog_versions" PRIMARY KEY ("version")
);

CREATE INDEX "idx_report_catalog_versions_last_seen" ON "report_catalog_versions"("last_seen_at");

COMMENT ON TABLE report_catalog_versions IS 'Report catalog versions of the running report workers, for the catalog check';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('report_catalog_versions', 'Report catalog versions the running workers were built with; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
