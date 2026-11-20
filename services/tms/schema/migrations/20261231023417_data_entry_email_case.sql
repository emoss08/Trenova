UPDATE "data_entry_controls"
SET "email_case" = 'Lower',
    "version" = "version" + 1,
    "updated_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
WHERE "email_case" = 'TitleCase';

ALTER TABLE "data_entry_controls"
    ADD CONSTRAINT "ck_data_entry_controls_email_case"
    CHECK ("email_case" IN ('AsEntered', 'Upper', 'Lower'));
