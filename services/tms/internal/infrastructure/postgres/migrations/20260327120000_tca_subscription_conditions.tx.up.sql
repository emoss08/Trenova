ALTER TABLE "tca_subscriptions"
    ADD COLUMN "conditions"       JSONB        NOT NULL DEFAULT '[]',
    ADD COLUMN "condition_match"  VARCHAR(10)  NOT NULL DEFAULT 'all',
    ADD COLUMN "watched_columns"  JSONB        NOT NULL DEFAULT '[]',
    ADD COLUMN "custom_title"     VARCHAR(500) NOT NULL DEFAULT '',
    ADD COLUMN "custom_message"   TEXT         NOT NULL DEFAULT '',
    ADD COLUMN "topic"            VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN "priority"         VARCHAR(20)  NOT NULL DEFAULT 'medium';
