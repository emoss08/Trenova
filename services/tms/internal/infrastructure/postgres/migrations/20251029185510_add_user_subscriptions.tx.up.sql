CREATE TYPE subscription_notification_type_enum AS ENUM(
    'Email',
    'InApp',
    'Both'
);

CREATE TABLE IF NOT EXISTS "user_subscriptions"(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "notification_type" subscription_notification_type_enum NOT NULL,
    "entity_type" varchar(100) NOT NULL,
    "entity_id" varchar(100) NOT NULL,
    "expires_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_user_subscriptions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_user_subscriptions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_subscriptions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_subscriptions_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_subscriptions_user_entity" ON "user_subscriptions"("user_id", "entity_type", "entity_id", "organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_user_subscriptions_created_updated" ON "user_subscriptions"("created_at", "updated_at");

