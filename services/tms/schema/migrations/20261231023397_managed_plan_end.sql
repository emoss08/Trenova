-- An owner can set a managed instance's plan to end at the close of the period already paid for.
-- The control plane says so with every feature list until the plan is kept, and Trenova warns the
-- instance's administrators: once when the end is set, once if it is kept, and once three days
-- before it. The end and what was announced are kept on the one feature state row, decided in the
-- transaction that stores the list, so repeated lists and retries announce nothing twice. Times
-- are Unix seconds, as everywhere else on this row.
ALTER TABLE "managed_feature_states"
    ADD COLUMN IF NOT EXISTS "plan_ends_at" bigint,
    ADD COLUMN IF NOT EXISTS "plan_end_announced" bigint,
    ADD COLUMN IF NOT EXISTS "plan_end_reminded" bigint;

ALTER TABLE "managed_feature_states"
    ADD CONSTRAINT "ck_managed_feature_states_plan_ends_at" CHECK ("plan_ends_at" IS NULL OR "plan_ends_at" > 0);

COMMENT ON COLUMN "managed_feature_states"."plan_ends_at" IS 'When the plan ends, in Unix seconds: the end of the period already paid for. Null while the plan is not set to end';

COMMENT ON COLUMN "managed_feature_states"."plan_end_announced" IS 'The plan end the administrators were told about, in Unix seconds; null when none is announced, so a kept plan is announced once';

COMMENT ON COLUMN "managed_feature_states"."plan_end_reminded" IS 'The plan end the administrators were reminded of three days before, in Unix seconds; equal to plan_ends_at once that reminder is sent or not needed';
