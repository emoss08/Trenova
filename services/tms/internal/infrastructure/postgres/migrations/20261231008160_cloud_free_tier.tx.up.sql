CREATE TABLE IF NOT EXISTS "organization_subscriptions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "plan_key" varchar(50) NOT NULL,
    "status" varchar(20) NOT NULL,
    "trial_ends_at" bigint NOT NULL,
    "read_only_until" bigint NOT NULL,
    "stripe_customer_id" varchar(255),
    "stripe_subscription_id" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_organization_subscriptions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_organization_subscriptions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_organization_subscriptions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_organization_subscriptions_status" CHECK ("status" IN ('trialing', 'active', 'read_only', 'expired')),
    CONSTRAINT "ck_organization_subscriptions_plan_key" CHECK (length(btrim("plan_key")) > 0),
    CONSTRAINT "ck_organization_subscriptions_read_only_after_trial" CHECK ("read_only_until" >= "trial_ends_at")
);

--bun:split
-- One subscription per organization: the plan resolver reads it by organization alone.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_organization_subscriptions_organization" ON "organization_subscriptions"("organization_id");

--bun:split
-- The hourly sweep moves trials past their end to read-only.
CREATE INDEX IF NOT EXISTS "idx_organization_subscriptions_trial_due" ON "organization_subscriptions"("trial_ends_at")
WHERE
    "status" = 'trialing';

--bun:split
-- The hourly sweep moves read-only organizations past their grace to expired.
CREATE INDEX IF NOT EXISTS "idx_organization_subscriptions_read_only_due" ON "organization_subscriptions"("read_only_until")
WHERE
    "status" = 'read_only';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_organization_subscriptions_stripe_subscription" ON "organization_subscriptions"("stripe_subscription_id")
WHERE
    "stripe_subscription_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE "organization_subscriptions" IS 'The plan a Trenova Cloud organization is on and where it is in its lifecycle. Only organizations created by cloud signup have a row; an organization without one is an unlimited internal organization.';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."plan_key" IS 'Plan defined in code (platformplan): free_demo today, paid plan keys once billing writes them';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."status" IS 'trialing, then read_only after trial_ends_at, then expired after read_only_until; active is reserved for paid plans';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."trial_ends_at" IS 'Unix seconds at which writes stop';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."read_only_until" IS 'Unix seconds at which the organization expires and is purged';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."stripe_customer_id" IS 'Reserved for paid billing';

--bun:split
COMMENT ON COLUMN "organization_subscriptions"."stripe_subscription_id" IS 'Reserved for paid billing';

--bun:split
CREATE TABLE IF NOT EXISTS "organization_onboarding"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL,
    "operation_type" varchar(20),
    "sample_data_loaded" boolean NOT NULL DEFAULT FALSE,
    "completed_at" bigint,
    "completed_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_organization_onboarding" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_organization_onboarding_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_organization_onboarding_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_organization_onboarding_status" CHECK ("status" IN ('pending', 'completed')),
    CONSTRAINT "ck_organization_onboarding_operation_type" CHECK ("operation_type" IS NULL OR "operation_type" IN ('asset', 'brokerage', 'both')),
    CONSTRAINT "ck_organization_onboarding_completion" CHECK (("status" = 'completed') = ("completed_at" IS NOT NULL)),
    CONSTRAINT "ck_organization_onboarding_completed_operation" CHECK ("status" <> 'completed' OR "operation_type" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_organization_onboarding_organization" ON "organization_onboarding"("organization_id");

--bun:split
COMMENT ON TABLE "organization_onboarding" IS 'Whether a Trenova Cloud organization has finished the first-run wizard, and what it told us about itself';

--bun:split
COMMENT ON COLUMN "organization_onboarding"."operation_type" IS 'asset, brokerage or both; sets the organization capability flags';

--bun:split
COMMENT ON COLUMN "organization_onboarding"."sample_data_loaded" IS 'Whether the wizard created the sample customers, locations, equipment and shipments';

--bun:split
CREATE TABLE IF NOT EXISTS "cloud_signups"(
    "id" varchar(100) NOT NULL,
    "email_address" varchar(255) NOT NULL,
    "email_normalized" varchar(255) NOT NULL,
    "name" varchar(255) NOT NULL,
    "company_name" varchar(255) NOT NULL,
    "password_hash" text NOT NULL,
    "token_hash" varchar(64) NOT NULL,
    "status" varchar(20) NOT NULL,
    "client_ip" varchar(64),
    "user_agent" varchar(512),
    "attempts" integer NOT NULL DEFAULT 0,
    "rejection_reason" varchar(100),
    "expires_at" bigint NOT NULL,
    "verified_at" bigint,
    "provisioned_organization_id" varchar(100),
    "provisioned_business_unit_id" varchar(100),
    "provisioned_user_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_cloud_signups" PRIMARY KEY ("id"),
    CONSTRAINT "ck_cloud_signups_status" CHECK ("status" IN ('pending', 'provisioned', 'expired', 'rejected')),
    CONSTRAINT "ck_cloud_signups_attempts" CHECK ("attempts" >= 0),
    CONSTRAINT "ck_cloud_signups_token_hash" CHECK ("token_hash" ~ '^[0-9a-f]{64}$'),
    CONSTRAINT "ck_cloud_signups_provisioned" CHECK ("status" <> 'provisioned' OR ("provisioned_organization_id" IS NOT NULL AND "provisioned_business_unit_id" IS NOT NULL AND "provisioned_user_id" IS NOT NULL AND "verified_at" IS NOT NULL))
);

--bun:split
-- One open request per address: a second signup for the same person replaces the token
-- on the pending row instead of adding another.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_cloud_signups_pending_email" ON "cloud_signups"("email_normalized")
WHERE
    "status" = 'pending';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_cloud_signups_token_hash" ON "cloud_signups"("token_hash");

--bun:split
-- The daily signup cap counts verified signups since the start of the UTC day.
CREATE INDEX IF NOT EXISTS "idx_cloud_signups_provisioned_verified" ON "cloud_signups"("verified_at")
WHERE
    "status" = 'provisioned';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_cloud_signups_pending_expires" ON "cloud_signups"("expires_at")
WHERE
    "status" = 'pending';

--bun:split
COMMENT ON TABLE "cloud_signups" IS 'Trenova Cloud signup requests waiting for email verification. Nothing is provisioned until the token is verified. Read and written only under the system scope; tenants may see only the request that provisioned them.';

--bun:split
COMMENT ON COLUMN "cloud_signups"."email_normalized" IS 'Lower-cased address with Gmail dots and +tags removed, for uniqueness';

--bun:split
COMMENT ON COLUMN "cloud_signups"."password_hash" IS 'argon2id hash of the chosen password; copied to the owner user at provisioning';

--bun:split
COMMENT ON COLUMN "cloud_signups"."token_hash" IS 'Hex SHA-256 of the emailed verification token; the token itself is never stored';

--bun:split
COMMENT ON COLUMN "cloud_signups"."attempts" IS 'Verification and resend attempts against this request';

--bun:split
SELECT trenova_rls.apply_policy(
    'public.cloud_signups',
    $expr$provisioned_organization_id = (SELECT trenova_rls.org_id())
        AND provisioned_business_unit_id = (SELECT trenova_rls.bu_id())$expr$
);

--bun:split
SELECT trenova_rls.reconcile();
