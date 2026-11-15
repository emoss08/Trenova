-- A dedicated instance run by Trenova Cloud learns from the control plane, through the operator,
-- which features its customer bought. One row, replaced whenever the operator sends the list and
-- read by every API and worker process, so a change reaches all of them within a cache interval.
-- Features on the base plan are always on and need not be listed.
CREATE TABLE IF NOT EXISTS "managed_feature_states"(
    "id" varchar(20) NOT NULL,
    "features" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "received_at" bigint NOT NULL,
    CONSTRAINT "pk_managed_feature_states" PRIMARY KEY ("id"),
    CONSTRAINT "ck_managed_feature_states_singleton" CHECK ("id" = 'current'),
    CONSTRAINT "ck_managed_feature_states_features" CHECK (jsonb_typeof("features") = 'array')
);

COMMENT ON TABLE "managed_feature_states" IS 'The features the control plane last said a managed instance may use, beyond the base plan';

COMMENT ON COLUMN "managed_feature_states"."features" IS 'The feature keys the customer bought, as the TMS feature catalog names them';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('managed_feature_states', 'The instance-wide list of purchased features the control plane sets for a managed instance; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
