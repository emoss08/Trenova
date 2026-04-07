--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS billing_queue_filter_presets(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "filters" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "is_default" boolean NOT NULL DEFAULT false,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_billing_queue_filter_presets" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_queue_filter_presets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_filter_presets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_filter_presets_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_billing_queue_filter_presets_user ON billing_queue_filter_presets("user_id", "organization_id", "business_unit_id");
