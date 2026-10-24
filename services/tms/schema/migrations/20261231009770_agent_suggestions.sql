CREATE TABLE IF NOT EXISTS "agent_suggestions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "suggested_agent_id" varchar(100) NOT NULL,
    "current_agent_id" varchar(100),
    "surface" varchar(20) NOT NULL,
    "outcome" varchar(20) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_suggestions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_suggestions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suggestions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suggestions_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suggestions_suggested_agent" FOREIGN KEY ("suggested_agent_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suggestions_current_agent" FOREIGN KEY ("current_agent_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("current_agent_id"),
    CONSTRAINT "ck_agent_suggestions_surface" CHECK ("surface" IN ('Home', 'Conversation')),
    CONSTRAINT "ck_agent_suggestions_outcome" CHECK ("outcome" IN ('Shown', 'Switched', 'SentThrough', 'AutoSwitched', 'Kept', 'PassedOver')),
    CONSTRAINT "ck_agent_suggestions_other_agent" CHECK ("current_agent_id" IS NULL OR "current_agent_id" <> "suggested_agent_id")
);

CREATE INDEX IF NOT EXISTS "idx_agent_suggestions_created" ON "agent_suggestions"("organization_id", "business_unit_id", "created_at", "suggested_agent_id");

COMMENT ON TABLE "agent_suggestions" IS 'Each agent the Desk named, or moved to, for a question a person was asking, and what the person did with it; read by AI analytics';

SELECT
    trenova_rls.reconcile();
