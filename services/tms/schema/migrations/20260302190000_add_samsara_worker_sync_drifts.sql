CREATE TABLE IF NOT EXISTS "samsara_worker_sync_drifts"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "worker_id" character varying(100) NOT NULL,
    "drift_type" character varying(64) NOT NULL,
    "worker_name" character varying(255) NOT NULL,
    "message" text NOT NULL,
    "local_external_id" text,
    "remote_driver_id" text,
    "detected_at" bigint NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "drift_type")
);

CREATE INDEX IF NOT EXISTS idx_worker_sync_drifts_tenant_detected_at
    ON "samsara_worker_sync_drifts" ("organization_id", "business_unit_id", "detected_at" DESC);
