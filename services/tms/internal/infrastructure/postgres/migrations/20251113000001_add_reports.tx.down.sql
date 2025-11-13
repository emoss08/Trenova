DROP TRIGGER IF EXISTS reports_search_update ON "reports";

DROP FUNCTION IF EXISTS reports_search_trigger();

DROP INDEX IF EXISTS idx_reports_search_vector;

DROP INDEX IF EXISTS idx_reports_resource_type;

DROP INDEX IF EXISTS idx_reports_expires_at;

DROP INDEX IF EXISTS idx_reports_created_at;

DROP INDEX IF EXISTS idx_reports_status;

DROP INDEX IF EXISTS idx_reports_user;

DROP INDEX IF EXISTS idx_reports_org_bu;

DROP TABLE IF EXISTS "reports";

DROP TYPE IF EXISTS report_status_enum;

DROP TYPE IF EXISTS report_delivery_method_enum;

DROP TYPE IF EXISTS report_format_enum;
