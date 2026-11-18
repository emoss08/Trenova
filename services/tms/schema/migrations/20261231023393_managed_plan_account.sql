-- The control plane sends two more things with a managed instance's feature list: the address of
-- the instance's plan page in the Trenova console, where add-ons are bought, and the add-ons whose
-- removal is scheduled, with the day each ends. Both live on the one feature state row, replaced
-- with it. The plan page links to the console and marks an add-on that is ending.
ALTER TABLE "managed_feature_states"
    ADD COLUMN IF NOT EXISTS "plan_url" text,
    ADD COLUMN IF NOT EXISTS "addon_endings" jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE "managed_feature_states"
    ADD CONSTRAINT "ck_managed_feature_states_plan_url" CHECK ("plan_url" IS NULL OR char_length("plan_url") BETWEEN 1 AND 2048);

ALTER TABLE "managed_feature_states"
    ADD CONSTRAINT "ck_managed_feature_states_addon_endings" CHECK (jsonb_typeof("addon_endings") = 'array');

COMMENT ON COLUMN "managed_feature_states"."plan_url" IS 'The absolute address of the instance''s plan page in the Trenova console, where add-ons are bought; null until the control plane sends one';

COMMENT ON COLUMN "managed_feature_states"."addon_endings" IS 'The add-ons whose removal is scheduled, each with its add-on key and the Unix time it ends';

-- A change to the plan now also records an add-on that was set to end (with its day) and an
-- add-on whose scheduled end was called off while it stays on the plan.
ALTER TABLE "managed_feature_changes"
    ADD COLUMN IF NOT EXISTS "ending" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "kept" jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE "managed_feature_changes"
    ADD CONSTRAINT "ck_managed_feature_changes_ending" CHECK (jsonb_typeof("ending") = 'array');

ALTER TABLE "managed_feature_changes"
    ADD CONSTRAINT "ck_managed_feature_changes_kept" CHECK (jsonb_typeof("kept") = 'array');

COMMENT ON COLUMN "managed_feature_changes"."ending" IS 'The add-ons the change set to end, each with its add-on key and the Unix time it ends';

COMMENT ON COLUMN "managed_feature_changes"."kept" IS 'The add-on keys whose scheduled end the change called off; they stay on the plan';
