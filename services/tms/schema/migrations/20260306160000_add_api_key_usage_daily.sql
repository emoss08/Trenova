CREATE TABLE "api_key_usage_daily" (
    "api_key_id" varchar(100) NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    "organization_id" varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "usage_date" date NOT NULL,
    "request_count" bigint NOT NULL DEFAULT 0,
    PRIMARY KEY ("api_key_id", "usage_date")
);

CREATE INDEX "idx_api_key_usage_daily_org_bu_date"
    ON "api_key_usage_daily"("organization_id", "business_unit_id", "usage_date");
