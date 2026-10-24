-- keel bumps a version per tenant and table after a write commits; report result caches key on
-- them. The application role has no CREATE, so the table is created here with keel's definition.
CREATE TABLE IF NOT EXISTS "keel_data_versions" (
    "tenant_key" text COLLATE "C" NOT NULL,
    "table_name" text COLLATE "C" NOT NULL,
    "version" bigint NOT NULL,
    "changed_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("tenant_key", "table_name")
) WITH (fillfactor = 70);

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('keel_data_versions', 'Cache invalidation counters keyed by tenant')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
