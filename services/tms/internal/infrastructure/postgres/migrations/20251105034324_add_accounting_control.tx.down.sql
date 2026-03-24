DROP TRIGGER IF EXISTS accounting_controls_update_timestamp_trigger ON accounting_controls;

DROP FUNCTION IF EXISTS accounting_controls_update_timestamp() CASCADE;

DROP TABLE IF EXISTS "accounting_controls" CASCADE;

DROP TYPE IF EXISTS "journal_entry_criteria_enum";

DROP TYPE IF EXISTS "reconciliation_threshold_action_enum";

DROP TYPE IF EXISTS "revenue_recognition_enum";

DROP TYPE IF EXISTS "expense_recognition_enum";

