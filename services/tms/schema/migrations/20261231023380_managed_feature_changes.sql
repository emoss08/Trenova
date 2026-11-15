-- Every change the control plane makes to a managed instance's feature list, so an administrator
-- can see when an add-on joined or left the plan. Written in the same transaction as the new list
-- in managed_feature_states; the first list an instance receives is its starting plan. Only the
-- most recent 200 changes are kept, pruned in the same write.
CREATE TABLE IF NOT EXISTS "managed_feature_changes"(
    "id" varchar(100) NOT NULL,
    "added" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "removed" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "starting" boolean NOT NULL DEFAULT FALSE,
    "changed_at" bigint NOT NULL,
    CONSTRAINT "pk_managed_feature_changes" PRIMARY KEY ("id"),
    CONSTRAINT "ck_managed_feature_changes_added" CHECK (jsonb_typeof("added") = 'array'),
    CONSTRAINT "ck_managed_feature_changes_removed" CHECK (jsonb_typeof("removed") = 'array')
);

COMMENT ON TABLE "managed_feature_changes" IS 'The changes the control plane made to a managed instance''s feature list, newest kept, at most 200';

COMMENT ON COLUMN "managed_feature_changes"."added" IS 'The feature keys the change turned on, as the TMS feature catalog names them';

COMMENT ON COLUMN "managed_feature_changes"."removed" IS 'The feature keys the change turned off; their data is kept';

COMMENT ON COLUMN "managed_feature_changes"."starting" IS 'True for the first list the instance received: its starting plan, not a change';

CREATE INDEX IF NOT EXISTS "idx_managed_feature_changes_changed_at"
    ON "managed_feature_changes"("changed_at" DESC, "id" DESC);

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('managed_feature_changes', 'The instance-wide history of the feature list the control plane sets for a managed instance; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
