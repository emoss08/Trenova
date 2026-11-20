-- The plan a managed install answers is held per scope instead of once. A dedicated instance has
-- one scope, the instance ('instance'); a shared server has one per organization, keyed by the
-- organization's id. The feature list, its history, the plan URL, the plan's end and the add-on
-- endings, and the usage state each carry the scope they belong to. A scope is also active or
-- suspended: a suspended organization on a shared server has no features at all. These stay
-- global tables, read and written only by the managed services under system scope, because they
-- are the control plane's entitlement metadata and hold no customer data. A dedicated instance's
-- one row becomes its instance scope.
ALTER TABLE "managed_feature_states"
    DROP CONSTRAINT IF EXISTS "ck_managed_feature_states_singleton";

ALTER TABLE "managed_feature_states"
    RENAME COLUMN "id" TO "scope_key";

ALTER TABLE "managed_feature_states"
    ALTER COLUMN "scope_key" TYPE varchar(100),
    ADD COLUMN IF NOT EXISTS "scope_business_unit_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "lifecycle" varchar(20) NOT NULL DEFAULT 'active',
    ALTER COLUMN "received_at" SET DEFAULT 0;

UPDATE "managed_feature_states"
SET "scope_key" = 'instance'
WHERE "scope_key" = 'current';

ALTER TABLE "managed_feature_states"
    ADD CONSTRAINT "ck_managed_feature_states_lifecycle" CHECK ("lifecycle" IN ('active', 'suspended')),
    ADD CONSTRAINT "ck_managed_feature_states_scope" CHECK (
        ("scope_key" = 'instance' AND "scope_business_unit_id" IS NULL AND "lifecycle" = 'active')
        OR ("scope_key" <> 'instance' AND "scope_business_unit_id" IS NOT NULL)
    );

COMMENT ON TABLE "managed_feature_states" IS 'The plan the control plane last set for each managed scope: the instance on a dedicated install, each organization on a shared server';

COMMENT ON COLUMN "managed_feature_states"."scope_key" IS 'Whose plan this is: instance on a dedicated install, the organization''s id on a shared server';

COMMENT ON COLUMN "managed_feature_states"."scope_business_unit_id" IS 'The business unit of the organization the scope names; null for the instance';

COMMENT ON COLUMN "managed_feature_states"."lifecycle" IS 'active, or suspended: a suspended organization has no features, base included; the instance is always active';

COMMENT ON COLUMN "managed_feature_states"."received_at" IS 'When the control plane last sent this scope''s feature list, in Unix seconds; 0 while it has sent none (a scope suspended before its first list)';

ALTER TABLE "managed_feature_changes"
    ADD COLUMN IF NOT EXISTS "scope_key" varchar(100);

UPDATE "managed_feature_changes"
SET "scope_key" = 'instance'
WHERE "scope_key" IS NULL;

ALTER TABLE "managed_feature_changes"
    ALTER COLUMN "scope_key" SET NOT NULL;

COMMENT ON TABLE "managed_feature_changes" IS 'The changes the control plane made to each managed scope''s feature list, the newest 200 kept per scope';

COMMENT ON COLUMN "managed_feature_changes"."scope_key" IS 'The scope whose plan changed: instance, or an organization''s id on a shared server';

ALTER TABLE "managed_usage_states"
    DROP CONSTRAINT IF EXISTS "ck_managed_usage_states_singleton";

ALTER TABLE "managed_usage_states"
    RENAME COLUMN "id" TO "scope_key";

ALTER TABLE "managed_usage_states"
    ALTER COLUMN "scope_key" TYPE varchar(100);

UPDATE "managed_usage_states"
SET "scope_key" = 'instance'
WHERE "scope_key" = 'current';

COMMENT ON TABLE "managed_usage_states" IS 'What the control plane last allowed each managed scope: the usage window, overage and each meter against its allowance and ceiling';

COMMENT ON COLUMN "managed_usage_states"."scope_key" IS 'Whose usage this is: instance on a dedicated install, the organization''s id on a shared server';

COMMENT ON COLUMN "managed_usage_states"."meters" IS 'Each metered action the control plane limits, by TMS meter key: quantity in the window, allowance, hard cap, ceiling (past which it stops whatever the overage setting; 0 for none) and whether it is stopped';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES
    ('managed_feature_states', 'The plan the control plane sets for each managed scope; entitlement metadata read and written only by the managed services under system scope, no customer data'),
    ('managed_feature_changes', 'The history of the plan the control plane sets for each managed scope; entitlement metadata, no customer data'),
    ('managed_usage_states', 'The usage state the control plane sets for each managed scope; entitlement metadata, no customer data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
