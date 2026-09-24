CREATE TABLE IF NOT EXISTS "agent_extensions" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "type" VARCHAR(50) NOT NULL,
    "enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    "availability" VARCHAR(20) NOT NULL DEFAULT 'SelectedAgents',
    "configuration" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "enabled_by_id" VARCHAR(100),
    "enabled_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_extensions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_extensions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extensions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extensions_enabled_by" FOREIGN KEY ("enabled_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "uq_agent_extensions_tenant_type" UNIQUE ("organization_id", "business_unit_id", "type"),
    CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa')),
    CONSTRAINT "ck_agent_extensions_availability" CHECK ("availability" IN ('AllAgents', 'SelectedAgents'))
);

COMMENT ON TABLE "agent_extensions" IS 'Capabilities an organization turns on for its agents only, such as web research, each with its own credentials and settings';

COMMENT ON COLUMN "agent_extensions"."availability" IS 'AllAgents gives every agent in the organization the extension''s tools; SelectedAgents offers them only to agents an administrator adds them to';

COMMENT ON COLUMN "agent_extensions"."configuration" IS 'Settings keyed by field; sensitive values are encrypted and bound to the tenant';

--bun:split

CREATE TABLE IF NOT EXISTS "agent_extension_usage_daily" (
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "extension_type" VARCHAR(50) NOT NULL,
    "day" INTEGER NOT NULL,
    "requests" INTEGER NOT NULL DEFAULT 0,
    "failures" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(19, 6) NOT NULL DEFAULT 0,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_extension_usage_daily" PRIMARY KEY ("organization_id", "business_unit_id", "extension_type", "day"),
    CONSTRAINT "fk_agent_extension_usage_daily_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extension_usage_daily_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_extension_usage_daily_requests" CHECK ("requests" >= 0 AND "failures" >= 0 AND "cost_usd" >= 0)
);

COMMENT ON TABLE "agent_extension_usage_daily" IS 'Requests each extension made to its provider per organization and UTC day (YYYYMMDD), with failures and the provider-reported cost; the daily request limit is enforced against it';
