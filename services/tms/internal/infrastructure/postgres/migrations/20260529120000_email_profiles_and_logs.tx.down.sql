DROP TABLE IF EXISTS email_suppressions;
DROP TABLE IF EXISTS email_events;
DROP TABLE IF EXISTS email_message_attachments;
DROP TABLE IF EXISTS email_messages;
DROP TABLE IF EXISTS email_profile_assignments;
DROP TYPE IF EXISTS email_suppression_reason_enum;
DROP TYPE IF EXISTS email_event_type_enum;
DROP TYPE IF EXISTS email_message_status_enum;
DROP TYPE IF EXISTS email_purpose_enum;

ALTER TABLE email_profiles
    ALTER COLUMN auth_type DROP DEFAULT,
    ALTER COLUMN encryption_type DROP DEFAULT;
