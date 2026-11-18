-- How many days a managed instance's final backup is kept after the instance is taken down is a
-- setting of the control plane, not a fixed number. The control plane sends it with every feature
-- list, and Trenova says it wherever it tells people what happens when the plan ends: the plan
-- page's notice and the notifications about the plan's end. Until the control plane says
-- otherwise it is 30, the control plane's default.
ALTER TABLE "managed_feature_states"
    ADD COLUMN IF NOT EXISTS "final_backup_days" integer NOT NULL DEFAULT 30;

ALTER TABLE "managed_feature_states"
    ADD CONSTRAINT "ck_managed_feature_states_final_backup_days" CHECK ("final_backup_days" BETWEEN 1 AND 3650);

COMMENT ON COLUMN "managed_feature_states"."final_backup_days" IS 'How many days the instance''s final backup is kept after the instance is taken down, as the control plane last said; 30 until it says otherwise';
