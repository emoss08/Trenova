-- The tenant archive (internal/cloud): one company's rows and documents taken out of a shared
-- server, put into another install, restored in place, and taken nightly.

-- An archive load copies a tenant's rows as they were. Three insert triggers would rewrite them:
-- two stamp updated_at with the load's clock and one fills an audit row's actor name from the
-- users loaded beside it. A load marks its own transaction with trenova.archive_load = on, and
-- only a session of a role that already bypasses row-level security (the system and migrator
-- roles) is believed, so a tenant session that sets the variable changes nothing.
CREATE OR REPLACE FUNCTION public.trenova_archive_loading()
    RETURNS boolean
    LANGUAGE sql
    STABLE
    SET search_path TO 'pg_catalog', 'pg_temp'
AS $$
    SELECT COALESCE(current_setting('trenova.archive_load', true), '') = 'on'
        AND pg_has_role(current_user, 'trenova_rls_bypass', 'MEMBER')
$$;

COMMENT ON FUNCTION public.trenova_archive_loading() IS 'True inside a tenant archive load run by a role that bypasses row-level security; history-stamping triggers leave the archived values alone';

CREATE OR REPLACE FUNCTION public.locations_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF public.trenova_archive_loading() THEN
        RETURN NEW;
    END IF;
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.shipment_holds_set_updated_at()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF public.trenova_archive_loading() THEN
        RETURN NEW;
    END IF;
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.audit_entries_actor_name()
    RETURNS trigger
    LANGUAGE plpgsql
    SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
    IF public.trenova_archive_loading() THEN
        RETURN NEW;
    END IF;

    IF NEW."actor_name" IS NOT NULL AND btrim(NEW."actor_name") <> '' THEN
        RETURN NEW;
    END IF;

    IF NEW."user_id" IS NOT NULL THEN
        SELECT left(COALESCE(NULLIF(btrim(u."name"), ''), u."username"), 255)
        INTO NEW."actor_name"
        FROM public.users u
        WHERE u."id" = NEW."user_id";
    ELSIF NEW."api_key_id" IS NOT NULL THEN
        SELECT left(k."name", 255)
        INTO NEW."actor_name"
        FROM public.api_keys k
        WHERE k."id" = NEW."api_key_id";
    ELSIF NEW."principal_type" = 'system' THEN
        NEW."actor_name" := 'System';
    END IF;

    RETURN NEW;
END
$function$;

-- An imported tenant's audit trail reaches back before this install's first audit partition. The
-- load asks for the months it needs before it begins; the function creates empty monthly
-- partitions and reads no tenant's rows, so it runs as its owner for the roles that bypass
-- row-level security, which own no table.
ALTER FUNCTION audit_partitions.ensure_partitions(bigint, integer) SECURITY DEFINER;

REVOKE ALL ON FUNCTION audit_partitions.ensure_partitions(bigint, integer) FROM PUBLIC;

GRANT USAGE ON SCHEMA audit_partitions TO trenova_rls_bypass;

GRANT EXECUTE ON FUNCTION audit_partitions.ensure_partitions(bigint, integer) TO trenova_rls_bypass;

ALTER TABLE "cloud_tenants"
    ADD COLUMN IF NOT EXISTS "last_restore_command_id" varchar(36),
    ADD COLUMN IF NOT EXISTS "last_restore_state" varchar(20),
    ADD COLUMN IF NOT EXISTS "last_restore_at" bigint,
    ADD COLUMN IF NOT EXISTS "last_restore_reason" varchar(500),
    ADD COLUMN IF NOT EXISTS "imported_at" bigint;

ALTER TABLE "cloud_tenants" DROP CONSTRAINT IF EXISTS "ck_cloud_tenants_state";

ALTER TABLE "cloud_tenants" ADD CONSTRAINT "ck_cloud_tenants_state"
    CHECK ("state" IN ('applying', 'active', 'suspended', 'restoring', 'removing', 'removed'));

ALTER TABLE "cloud_tenants" ADD CONSTRAINT "ck_cloud_tenants_restore"
    CHECK ("last_restore_state" IS NULL OR "last_restore_state" IN ('running', 'done', 'failed'));

COMMENT ON COLUMN "cloud_tenants"."last_restore_state" IS 'running, done or failed: the newest restore command of the tenant';

COMMENT ON COLUMN "cloud_tenants"."imported_at" IS 'When the tenant was made from an archive instead of from its first administrator';

ALTER TABLE "cloud_tenant_events" DROP CONSTRAINT IF EXISTS "ck_cloud_tenant_events_kind";

ALTER TABLE "cloud_tenant_events" ADD CONSTRAINT "ck_cloud_tenant_events_kind"
    CHECK ("kind" IN ('created', 'applied', 'refused', 'suspended', 'resumed', 'removing', 'removed',
        'archived', 'restoring', 'restored', 'imported'));

-- Every archive, restore and import the control plane asks a shared server for, by its command
-- id, so a repeated request answers the same command. The archive location's credentials and
-- archive key are held sealed with the install's key only while the command runs, and erased when
-- it settles; the command never carries them through Temporal.
CREATE TABLE IF NOT EXISTS "cloud_tenant_commands"(
    "command_id" varchar(36) NOT NULL,
    "instance_id" varchar(36) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "purpose" varchar(20),
    "state" varchar(20) NOT NULL,
    "location" varchar(1100) NOT NULL,
    "prefix" varchar(512) NOT NULL,
    "expect_sha256" varchar(64),
    "sealed_access" text,
    "progress" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "result_sha256" varchar(64),
    "result_size_bytes" bigint,
    "taken_at" bigint,
    "failure_code" varchar(20),
    "reason" varchar(500),
    "requested_at" bigint NOT NULL,
    "started_at" bigint,
    "finished_at" bigint,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_cloud_tenant_commands" PRIMARY KEY ("command_id"),
    CONSTRAINT "ck_cloud_tenant_commands_kind" CHECK ("kind" IN ('archive', 'restore', 'import')),
    CONSTRAINT "ck_cloud_tenant_commands_state" CHECK ("state" IN ('queued', 'running', 'done', 'failed')),
    CONSTRAINT "ck_cloud_tenant_commands_purpose" CHECK (
        ("kind" = 'archive') = ("purpose" IS NOT NULL)
        AND ("purpose" IS NULL OR "purpose" IN ('final', 'export', 'move', 'nightly'))
    ),
    CONSTRAINT "ck_cloud_tenant_commands_settled" CHECK (
        "state" IN ('queued', 'running') OR "sealed_access" IS NULL
    ),
    CONSTRAINT "ck_cloud_tenant_commands_progress" CHECK (jsonb_typeof("progress") = 'object')
);

CREATE INDEX IF NOT EXISTS "idx_cloud_tenant_commands_open"
    ON "cloud_tenant_commands"("instance_id", "kind")
    WHERE "state" IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS "idx_cloud_tenant_commands_instance"
    ON "cloud_tenant_commands"("instance_id", "requested_at" DESC);

COMMENT ON TABLE "cloud_tenant_commands" IS 'The archives, restores and imports a shared server ran for its tenants; credentials only while a command runs, sealed with the install key';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('cloud_tenant_commands', 'A shared server''s record of the archive, restore and import commands the control plane gave it; read and written only under system scope; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- The archive location the server takes nightly archives to, with short-lived credentials and the
-- region's archive key sealed with the install's key, as the tenant agent last delivered them.
CREATE TABLE IF NOT EXISTS "cloud_archive_grants"(
    "name" varchar(20) NOT NULL,
    "location" varchar(1100) NOT NULL,
    "sealed_access" text NOT NULL,
    "expires_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_cloud_archive_grants" PRIMARY KEY ("name"),
    CONSTRAINT "ck_cloud_archive_grants_name" CHECK ("name" IN ('nightly'))
);

COMMENT ON TABLE "cloud_archive_grants" IS 'Where and with what short-lived, sealed credentials a shared server takes its nightly tenant archives';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('cloud_archive_grants', 'A shared server''s nightly archive location and sealed credentials from the control plane; read and written only under system scope; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
