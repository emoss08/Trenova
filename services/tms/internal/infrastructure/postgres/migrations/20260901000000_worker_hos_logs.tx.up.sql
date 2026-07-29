CREATE TABLE IF NOT EXISTS "worker_hos_logs"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "worker_id" character varying(100) NOT NULL,
    "log_start_at" bigint NOT NULL,
    "duty_status" character varying(32) NOT NULL,
    "log_end_at" bigint,
    "remark" text,
    "provider" character varying(32) NOT NULL DEFAULT 'Samsara',
    "provider_vehicle_id" text,
    "received_at" bigint NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "log_start_at")
);

CREATE INDEX IF NOT EXISTS idx_worker_hos_logs_start_at
    ON "worker_hos_logs" ("log_start_at");
