-- Deleting a saved report keeps its runs: the run forgets the report it came from and keeps the
-- revision it ran, its status, timing and who ran it. Before, the key nulled the run's tenant
-- columns along with the report and the delete failed on any report that had runs.
ALTER TABLE "report_runs"
    DROP CONSTRAINT "fk_report_runs_definition",
    ADD CONSTRAINT "fk_report_runs_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("definition_id");

-- A run is bound to a saved report, the revision of one it ran, or a canned report.
ALTER TABLE "report_runs"
    DROP CONSTRAINT "check_report_runs_source",
    ADD CONSTRAINT "check_report_runs_source" CHECK ("definition_id" IS NOT NULL OR "revision_id" IS NOT NULL OR "canned_key" IS NOT NULL);
