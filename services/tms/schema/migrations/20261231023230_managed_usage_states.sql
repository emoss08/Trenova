-- A dedicated instance run by Trenova Cloud learns from the control plane, through the operator,
-- what may continue: the open usage window, whether overage is billed, and each metered action's
-- quantity against its allowance. One row, replaced on every report and read by every API and
-- worker process, so a stop reaches all of them within a cache interval.
CREATE TABLE IF NOT EXISTS "managed_usage_states"(
    "id" varchar(20) NOT NULL,
    "window_start" bigint,
    "window_end" bigint,
    "overage" boolean NOT NULL DEFAULT FALSE,
    "meters" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "received_at" bigint NOT NULL,
    CONSTRAINT "pk_managed_usage_states" PRIMARY KEY ("id"),
    CONSTRAINT "ck_managed_usage_states_singleton" CHECK ("id" = 'current'),
    CONSTRAINT "ck_managed_usage_states_window" CHECK (("window_start" IS NULL) = ("window_end" IS NULL)
        AND ("window_end" IS NULL OR "window_end" > "window_start")),
    CONSTRAINT "ck_managed_usage_states_meters" CHECK (jsonb_typeof("meters") = 'array')
);

COMMENT ON TABLE "managed_usage_states" IS 'What the control plane last allowed a managed instance: the usage window, overage and each meter against its allowance';

COMMENT ON COLUMN "managed_usage_states"."meters" IS 'Each metered action the control plane limits, by TMS meter key: quantity in the window, allowance, hard cap and whether it is stopped';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('managed_usage_states', 'The instance-wide usage state the control plane sets for a managed instance; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
