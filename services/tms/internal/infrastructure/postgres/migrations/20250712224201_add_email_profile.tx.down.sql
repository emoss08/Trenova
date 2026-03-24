--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TRIGGER IF EXISTS email_profiles_default_check ON email_profiles;

DROP TRIGGER IF EXISTS email_templates_search_update ON email_templates;

--bun:split
DROP FUNCTION IF EXISTS ensure_single_default_email_profile();

DROP FUNCTION IF EXISTS update_email_template_search_vector();

--bun:split
DROP TABLE IF EXISTS email_logs;

DROP TABLE IF EXISTS email_queue;

DROP TABLE IF EXISTS email_templates;

DROP TABLE IF EXISTS email_profiles;

--bun:split
DROP TYPE IF EXISTS email_bounce_type_enum;

DROP TYPE IF EXISTS email_log_status_enum;

DROP TYPE IF EXISTS email_queue_status_enum;

DROP TYPE IF EXISTS email_priority_enum;

DROP TYPE IF EXISTS email_template_category_enum;

DROP TYPE IF EXISTS email_encryption_type_enum;

DROP TYPE IF EXISTS email_auth_type_enum;

DROP TYPE IF EXISTS email_provider_type_enum;

