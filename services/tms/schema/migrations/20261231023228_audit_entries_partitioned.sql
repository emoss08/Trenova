CREATE SCHEMA IF NOT EXISTS "audit_partitions";

REVOKE ALL ON SCHEMA "audit_partitions" FROM PUBLIC;

COMMENT ON SCHEMA "audit_partitions" IS 'Monthly partitions of audit_entries; reached only through the parent, so the application role holds no privilege here';

CREATE OR REPLACE FUNCTION "audit_retention_floor_days"("retention_class" TEXT)
    RETURNS INTEGER
    LANGUAGE sql
    IMMUTABLE
    PARALLEL SAFE
    SET search_path = pg_catalog, pg_temp
AS $$
    SELECT CASE "retention_class"
        WHEN 'Access' THEN 90
        WHEN 'Operational' THEN 730
        WHEN 'Security' THEN 730
        WHEN 'Financial' THEN 2555
        ELSE 3650
    END
$$;

COMMENT ON FUNCTION "audit_retention_floor_days"(TEXT) IS 'The fewest days an audit entry of a retention class is kept, whatever its organization chose';

CREATE TABLE IF NOT EXISTS "audit_legal_holds"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(200) NOT NULL,
    "reason" TEXT NOT NULL,
    "resource" VARCHAR(50),
    "resource_id" VARCHAR(100),
    "placed_by_id" VARCHAR(100) NOT NULL,
    "placed_at" BIGINT NOT NULL,
    "released_by_id" VARCHAR(100),
    "released_at" BIGINT,
    "release_reason" TEXT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_audit_legal_holds" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_audit_legal_holds_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_legal_holds_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_legal_holds_name" CHECK (length(btrim("name")) > 0),
    CONSTRAINT "ck_audit_legal_holds_reason" CHECK (length(btrim("reason")) > 0),
    CONSTRAINT "ck_audit_legal_holds_scope" CHECK ("resource_id" IS NULL OR "resource" IS NOT NULL),
    CONSTRAINT "ck_audit_legal_holds_release" CHECK (
        ("released_at" IS NULL AND "released_by_id" IS NULL AND "release_reason" IS NULL)
        OR ("released_at" IS NOT NULL AND "released_by_id" IS NOT NULL AND length(btrim(coalesce("release_reason", ''))) > 0)
    )
);

CREATE INDEX IF NOT EXISTS "idx_audit_legal_holds_tenant_active" ON "audit_legal_holds"("organization_id", "business_unit_id") WHERE "released_at" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_audit_legal_holds_tenant_placed" ON "audit_legal_holds"("organization_id", "business_unit_id", "placed_at" DESC, "id" DESC);

COMMENT ON TABLE "audit_legal_holds" IS 'Holds that stop the retention sweep deleting an organization''s audit entries, all of them or one resource''s or record''s, until released';

CREATE OR REPLACE FUNCTION "audit_legal_holds_guard"()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF pg_trigger_depth() > 1 THEN
            RETURN OLD;
        END IF;
        RAISE EXCEPTION 'Legal holds are released, never deleted'
            USING ERRCODE = 'insufficient_privilege';
    END IF;

    IF OLD."released_at" IS NOT NULL THEN
        RAISE EXCEPTION 'A released legal hold cannot be changed'
            USING ERRCODE = 'insufficient_privilege';
    END IF;

    IF NEW."organization_id" <> OLD."organization_id"
        OR NEW."business_unit_id" <> OLD."business_unit_id"
        OR NEW."resource" IS DISTINCT FROM OLD."resource"
        OR NEW."resource_id" IS DISTINCT FROM OLD."resource_id"
        OR NEW."placed_by_id" <> OLD."placed_by_id"
        OR NEW."placed_at" <> OLD."placed_at"
    THEN
        RAISE EXCEPTION 'A legal hold''s scope and who placed it cannot be changed'
            USING ERRCODE = 'insufficient_privilege';
    END IF;

    RETURN NEW;
END
$$;

CREATE TRIGGER "trg_audit_legal_holds_guard"
    BEFORE UPDATE OR DELETE ON "audit_legal_holds"
    FOR EACH ROW
    EXECUTE FUNCTION "audit_legal_holds_guard"();

ALTER TABLE "audit_entries" RENAME TO "audit_entries_legacy";

DROP TRIGGER IF EXISTS "enforce_audit_append_only" ON "audit_entries_legacy";

CREATE TABLE "audit_entries"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "timestamp" BIGINT NOT NULL,
    "resource" VARCHAR(50) NOT NULL,
    "resource_id" VARCHAR(100) NOT NULL,
    "operation" VARCHAR(50) NOT NULL,
    "action" VARCHAR(100) NOT NULL,
    "category" VARCHAR(20) NOT NULL,
    "retention_class" VARCHAR(20) NOT NULL,
    "channel" VARCHAR(30) NOT NULL,
    "principal_type" VARCHAR(50) NOT NULL,
    "principal_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100),
    "api_key_id" VARCHAR(100),
    "actor_name" VARCHAR(255),
    "ai_run_id" VARCHAR(100),
    "entity_ref" VARCHAR(255),
    "correlation_id" VARCHAR(100),
    "trace_id" VARCHAR(64),
    "ip_address" VARCHAR(45),
    "user_agent" VARCHAR(255),
    "comment" TEXT,
    "changes" JSONB COMPRESSION lz4,
    "previous_state" JSONB COMPRESSION lz4,
    "current_state" JSONB COMPRESSION lz4,
    "metadata" JSONB COMPRESSION lz4,
    "sensitive_data" BOOLEAN NOT NULL DEFAULT FALSE,
    "critical" BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT "pk_audit_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id", "timestamp"),
    CONSTRAINT "fk_audit_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_entries_category" CHECK ("category" IN ('System', 'User', 'Security')),
    CONSTRAINT "ck_audit_entries_retention_class" CHECK ("retention_class" IN ('Access', 'Operational', 'Financial', 'Security')),
    CONSTRAINT "ck_audit_entries_channel" CHECK ("channel" IN ('web', 'api', 'portal', 'agent_conversation', 'agent_run', 'support', 'scim', 'job', 'cli', 'system')),
    CONSTRAINT "ck_audit_entries_action" CHECK ("action" ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    CONSTRAINT "ck_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key', 'system', 'agent')),
    CONSTRAINT "ck_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
        OR
        ("principal_type" = 'system' AND "user_id" IS NULL AND "api_key_id" IS NULL)
        OR
        ("principal_type" = 'agent' AND "api_key_id" IS NULL AND ("user_id" IS NULL OR "principal_id" <> "user_id"))
    ),
    CONSTRAINT "ck_audit_entries_changes" CHECK ("changes" IS NULL OR jsonb_typeof("changes") = 'object'),
    CONSTRAINT "ck_audit_entries_previous_state" CHECK ("previous_state" IS NULL OR jsonb_typeof("previous_state") = 'object'),
    CONSTRAINT "ck_audit_entries_current_state" CHECK ("current_state" IS NULL OR jsonb_typeof("current_state") = 'object'),
    CONSTRAINT "ck_audit_entries_metadata" CHECK ("metadata" IS NULL OR jsonb_typeof("metadata") = 'object')
)
PARTITION BY RANGE ("timestamp");

COMMENT ON TABLE "audit_entries" IS 'Append-only audit trail, partitioned by month of "timestamp"; written in the transaction of the change it records';

COMMENT ON COLUMN "audit_entries"."action" IS 'Stable dotted action code (resource.operation unless the writer named a more specific one), what alert rules and exports key on';

COMMENT ON COLUMN "audit_entries"."retention_class" IS 'Which of the organization''s audit retention periods applies; audit_retention_floor_days is the least any organization keeps it';

COMMENT ON COLUMN "audit_entries"."channel" IS 'How the change arrived: the web app, an API key, a portal, an agent in a conversation or on its own, a support session, SCIM, a background job, the CLI or the system';

COMMENT ON COLUMN "audit_entries"."actor_name" IS 'The name of who acted as it was when the entry was written, so renamed or removed principals still read correctly';

COMMENT ON COLUMN "audit_entries"."ai_run_id" IS 'The agent run or assistant turn whose tool call made the change, linking the entry to the AI audit trail';

CREATE INDEX "idx_audit_entries_tenant_time" ON "audit_entries"("organization_id", "business_unit_id", "timestamp" DESC, "id" DESC);

CREATE INDEX "idx_audit_entries_tenant_record" ON "audit_entries"("organization_id", "business_unit_id", "resource_id", "timestamp" DESC, "id" DESC);

CREATE INDEX "idx_audit_entries_tenant_principal" ON "audit_entries"("organization_id", "business_unit_id", "principal_id", "timestamp" DESC, "id" DESC);

CREATE INDEX "idx_audit_entries_tenant_resource" ON "audit_entries"("organization_id", "business_unit_id", "resource", "timestamp" DESC, "id" DESC);

CREATE INDEX "idx_audit_entries_tenant_correlation" ON "audit_entries"("organization_id", "business_unit_id", "correlation_id") WHERE "correlation_id" IS NOT NULL;

CREATE INDEX "idx_audit_entries_tenant_security" ON "audit_entries"("organization_id", "business_unit_id", "timestamp" DESC, "id" DESC) WHERE "category" = 'Security';

CREATE INDEX "idx_audit_entries_entity_ref_trgm" ON "audit_entries" USING gin ("entity_ref" gin_trgm_ops);

CREATE INDEX "idx_audit_entries_comment_trgm" ON "audit_entries" USING gin ("comment" gin_trgm_ops);

CREATE INDEX "idx_audit_entries_time_brin" ON "audit_entries" USING brin ("timestamp");

CREATE OR REPLACE FUNCTION "prevent_audit_modification"()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = public, pg_catalog, pg_temp
AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        RAISE EXCEPTION 'audit_entries cannot be truncated'
            USING ERRCODE = 'insufficient_privilege';
    END IF;

    IF TG_OP = 'DELETE' AND pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;

    IF TG_OP = 'DELETE'
        AND current_setting('trenova.audit_retention', true) = 'on'
        AND pg_has_role(current_user, 'trenova_rls_bypass', 'MEMBER')
        AND OLD."timestamp" < EXTRACT(epoch FROM current_timestamp - make_interval(days => audit_retention_floor_days(OLD."retention_class")))::BIGINT
        AND NOT EXISTS (
            SELECT 1
            FROM audit_legal_holds h
            WHERE h."organization_id" = OLD."organization_id"
                AND h."business_unit_id" = OLD."business_unit_id"
                AND h."released_at" IS NULL
                AND (h."resource" IS NULL OR (h."resource" = OLD."resource" AND (h."resource_id" IS NULL OR h."resource_id" = OLD."resource_id")))
        )
    THEN
        RETURN OLD;
    END IF;

    RAISE EXCEPTION 'Modifications are not allowed on audit_entries (append-only table)'
        USING ERRCODE = 'insufficient_privilege',
              HINT = 'Only the retention sweep may delete audit entries, never one younger than its retention class''s floor or under a legal hold';
END
$$;

CREATE TRIGGER "enforce_audit_append_only"
    BEFORE UPDATE OR DELETE ON "audit_entries"
    FOR EACH ROW
    EXECUTE FUNCTION "prevent_audit_modification"();

CREATE TRIGGER "enforce_audit_no_truncate"
    BEFORE TRUNCATE ON "audit_entries"
    FOR EACH STATEMENT
    EXECUTE FUNCTION "prevent_audit_modification"();

CREATE OR REPLACE FUNCTION "audit_entries_actor_name"()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
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
$$;

COMMENT ON FUNCTION "audit_entries_actor_name"() IS 'Names who an audit entry credits, as they were named when it was written';

CREATE TRIGGER "set_audit_entries_actor_name"
    BEFORE INSERT ON "audit_entries"
    FOR EACH ROW
    EXECUTE FUNCTION "audit_entries_actor_name"();

CREATE OR REPLACE FUNCTION "audit_partitions"."ensure_partitions"("p_from" BIGINT, "p_months_ahead" INTEGER)
    RETURNS INTEGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    first_month TIMESTAMP;
    last_month TIMESTAMP;
    month_start TIMESTAMP;
    partition_name TEXT;
    created INTEGER := 0;
BEGIN
    IF "p_months_ahead" IS NULL OR "p_months_ahead" < 1 OR "p_months_ahead" > 24 THEN
        RAISE EXCEPTION 'ensure_partitions creates between 1 and 24 months ahead, not %', "p_months_ahead";
    END IF;

    last_month := date_trunc('month', timezone('UTC', now())) + make_interval(months => "p_months_ahead");
    first_month := date_trunc('month', timezone('UTC', now()));
    IF "p_from" IS NOT NULL THEN
        first_month := LEAST(first_month, date_trunc('month', timezone('UTC', to_timestamp("p_from"))));
    END IF;

    PERFORM set_config('lock_timeout', '5s', true);

    month_start := first_month;
    WHILE month_start <= last_month LOOP
        partition_name := format('audit_entries_%s', to_char(month_start, 'YYYY_MM'));
        IF to_regclass(format('audit_partitions.%I', partition_name)) IS NULL THEN
            EXECUTE format(
                'CREATE TABLE audit_partitions.%I (LIKE public.audit_entries INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING STORAGE INCLUDING COMPRESSION)',
                partition_name
            );
            EXECUTE format(
                'ALTER TABLE public.audit_entries ATTACH PARTITION audit_partitions.%I FOR VALUES FROM (%s) TO (%s)',
                partition_name,
                EXTRACT(epoch FROM month_start AT TIME ZONE 'UTC')::BIGINT,
                EXTRACT(epoch FROM (month_start + interval '1 month') AT TIME ZONE 'UTC')::BIGINT
            );
            created := created + 1;
        END IF;
        month_start := month_start + interval '1 month';
    END LOOP;

    RETURN created;
END
$$;

COMMENT ON FUNCTION "audit_partitions"."ensure_partitions"(BIGINT, INTEGER) IS 'Creates every missing monthly audit_entries partition from the month of p_from (or this month) through p_months_ahead months ahead; trenova-db runs it as the table owner on every deploy';

CREATE OR REPLACE FUNCTION "audit_partitions"."drop_expired_partitions"("p_before" BIGINT)
    RETURNS INTEGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    candidate RECORD;
    dropped INTEGER := 0;
    held BOOLEAN;
BEGIN
    IF "p_before" > EXTRACT(epoch FROM now() - interval '3650 days')::BIGINT THEN
        RAISE EXCEPTION 'audit partitions are kept for at least 3650 days';
    END IF;

    PERFORM set_config('lock_timeout', '5s', true);

    FOR candidate IN
        SELECT c.oid::regclass AS target,
               c.relname AS name,
               (regexp_match(pg_get_expr(c.relpartbound, c.oid), 'TO \(''?(-?\d+)''?\)'))[1]::BIGINT AS upper_bound
        FROM pg_inherits i
        JOIN pg_class c ON c.oid = i.inhrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE i.inhparent = 'public.audit_entries'::regclass
            AND n.nspname = 'audit_partitions'
    LOOP
        CONTINUE WHEN candidate.upper_bound IS NULL OR candidate.upper_bound > "p_before";

        EXECUTE format(
            'SELECT EXISTS (SELECT 1 FROM %s e JOIN public.audit_legal_holds h ON h.organization_id = e.organization_id AND h.business_unit_id = e.business_unit_id AND h.released_at IS NULL AND (h.resource IS NULL OR (h.resource = e.resource AND (h.resource_id IS NULL OR h.resource_id = e.resource_id))))',
            candidate.target
        ) INTO held;
        CONTINUE WHEN held;

        EXECUTE format('ALTER TABLE public.audit_entries DETACH PARTITION %s', candidate.target);
        EXECUTE format('DROP TABLE %s', candidate.target);
        dropped := dropped + 1;
    END LOOP;

    RETURN dropped;
END
$$;

COMMENT ON FUNCTION "audit_partitions"."drop_expired_partitions"(BIGINT) IS 'Drops whole monthly audit_entries partitions that end before p_before (at least ten years ago) and hold no entry under a legal hold; trenova-db runs it as the table owner on every deploy';

REVOKE ALL ON FUNCTION "audit_partitions"."ensure_partitions"(BIGINT, INTEGER) FROM PUBLIC;

REVOKE ALL ON FUNCTION "audit_partitions"."drop_expired_partitions"(BIGINT) FROM PUBLIC;

SELECT "audit_partitions"."ensure_partitions"((SELECT min("timestamp") FROM "audit_entries_legacy"), 24);

INSERT INTO "audit_entries"(
    "id", "business_unit_id", "organization_id", "timestamp", "resource", "resource_id", "operation",
    "action", "category", "retention_class", "channel", "principal_type", "principal_id", "user_id",
    "api_key_id", "entity_ref", "correlation_id", "ip_address", "user_agent", "comment", "changes",
    "previous_state", "current_state", "metadata", "sensitive_data", "critical"
)
SELECT
    l."id",
    l."business_unit_id",
    l."organization_id",
    l."timestamp",
    l."resource",
    l."resource_id",
    l."operation",
    lower(regexp_replace(l."resource", '[^a-zA-Z0-9_]', '_', 'g')) || '.' || lower(regexp_replace(l."operation", '[^a-zA-Z0-9_]', '_', 'g')),
    CASE WHEN l."critical" AND l."category" = 'User' THEN 'Security' WHEN l."category" = 'User' THEN 'User' ELSE 'System' END,
    CASE
        WHEN l."operation" = 'read' THEN 'Access'
        WHEN l."critical" AND l."category" = 'User' THEN 'Security'
        WHEN l."resource" IN (
            'fuel_card', 'fuel_purchase', 'fuel_purchase_import', 'ifta_jurisdiction_mileage', 'ifta_return', 'ifta_tax_rate',
            'invoice', 'invoice_dispute', 'factoring_agreement', 'factoring_noa', 'factoring_submission', 'factoring_reserve',
            'factoring_remittance', 'billing_queue', 'invoice_run', 'accessorial_charge', 'charge_type', 'revenue_code',
            'formula_template', 'formula_library', 'fuel_surcharge_program', 'detention_policy', 'detention', 'rate_agreement',
            'rate_zone', 'rate_matrix', 'rate_quote', 'rate_simulation', 'rate_confirmation', 'accounting_control',
            'accounts_receivable', 'billing_control', 'costing_control', 'invoice_adjustment_control', 'account_type',
            'general_ledger_account', 'division_code', 'fiscal_year', 'fiscal_period', 'manual_journal', 'journal_reversal',
            'journal_entry', 'customer_payment', 'bank_receipt', 'bank_receipt_work_item', 'accounting_report',
            'accounting_integration', 'accounting_sync', 'driver_pay_profile', 'recurring_deduction', 'recurring_earning',
            'pay_code', 'pay_advance', 'escrow_account', 'driver_settlement', 'settlement_dispute', 'driver_expense',
            'settlement_control'
        ) THEN 'Financial'
        ELSE 'Operational'
    END,
    CASE l."principal_type"
        WHEN 'api_key' THEN 'api'
        WHEN 'agent' THEN 'agent_run'
        WHEN 'system' THEN 'system'
        ELSE 'web'
    END,
    l."principal_type",
    l."principal_id",
    l."user_id",
    l."api_key_id",
    left(NULLIF(btrim(COALESCE(
        l."current_state" ->> 'proNumber', l."current_state" ->> 'orderNumber', l."current_state" ->> 'invoiceNumber',
        l."current_state" ->> 'number', l."current_state" ->> 'code', l."current_state" ->> 'name',
        l."current_state" ->> 'username', l."current_state" ->> 'reference', l."current_state" ->> 'title',
        l."previous_state" ->> 'proNumber', l."previous_state" ->> 'number', l."previous_state" ->> 'code',
        l."previous_state" ->> 'name', l."metadata" ->> 'name', ''
    )), ''), 255),
    l."correlation_id",
    l."ip_address",
    l."user_agent",
    l."comment",
    NULLIF(l."changes", '{}'::jsonb),
    NULLIF(l."previous_state", '{}'::jsonb),
    NULLIF(l."current_state", '{}'::jsonb),
    NULLIF(l."metadata", '{}'::jsonb),
    l."sensitive_data",
    l."critical"
FROM "audit_entries_legacy" l;

DROP TABLE "audit_entries_legacy";

DROP TABLE IF EXISTS "audit_dlq_entries";

DROP TYPE IF EXISTS "audit_entry_category_enum";

ALTER TABLE "data_retention"
    DROP COLUMN IF EXISTS "audit_retention_period",
    ADD COLUMN "audit_access_retention_days" INTEGER NOT NULL DEFAULT 90,
    ADD COLUMN "audit_operational_retention_days" INTEGER NOT NULL DEFAULT 730,
    ADD COLUMN "audit_financial_retention_days" INTEGER NOT NULL DEFAULT 2555,
    ADD COLUMN "audit_security_retention_days" INTEGER NOT NULL DEFAULT 730,
    ADD CONSTRAINT "ck_data_retention_audit_access" CHECK ("audit_access_retention_days" BETWEEN 90 AND 3650),
    ADD CONSTRAINT "ck_data_retention_audit_operational" CHECK ("audit_operational_retention_days" BETWEEN 730 AND 3650),
    ADD CONSTRAINT "ck_data_retention_audit_financial" CHECK ("audit_financial_retention_days" BETWEEN 2555 AND 3650),
    ADD CONSTRAINT "ck_data_retention_audit_security" CHECK ("audit_security_retention_days" BETWEEN 730 AND 3650);

COMMENT ON COLUMN "data_retention"."audit_access_retention_days" IS 'Days audit entries of reads and downloads are kept; 90 to 3650';

COMMENT ON COLUMN "data_retention"."audit_operational_retention_days" IS 'Days audit entries of operational changes are kept; 730 to 3650';

COMMENT ON COLUMN "data_retention"."audit_financial_retention_days" IS 'Days audit entries of billing, accounting, rating and settlement changes are kept; 2555 (seven years) to 3650';

COMMENT ON COLUMN "data_retention"."audit_security_retention_days" IS 'Days security audit entries (sign-in, access and administrative changes, refused requests) are kept; 730 to 3650';
