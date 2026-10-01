DO $$
BEGIN
    BEGIN
        CREATE ROLE trenova_tenant NOLOGIN;
    EXCEPTION
        WHEN duplicate_object OR unique_violation THEN
            NULL;
    END;
    BEGIN
        CREATE ROLE trenova_rls_bypass NOLOGIN;
    EXCEPTION
        WHEN duplicate_object OR unique_violation THEN
            NULL;
    END;
END
$$;

--bun:split
DO $$
DECLARE
    existing record;
BEGIN
    FOR existing IN
        SELECT DISTINCT r.rolname
        FROM pg_roles r
        WHERE r.rolname NOT IN ('trenova_tenant', 'trenova_rls_bypass')
            AND NOT r.rolsuper
            AND (
                r.rolname = CURRENT_USER
                OR EXISTS (
                    SELECT 1
                    FROM pg_class c
                    JOIN pg_namespace n ON n.oid = c.relnamespace
                    WHERE n.nspname = 'public'
                        AND c.relkind IN ('r', 'p')
                        AND (
                            c.relowner = r.oid
                            OR EXISTS (SELECT 1 FROM aclexplode(c.relacl) a WHERE a.grantee = r.oid)
                        )
                )
            )
            AND NOT pg_has_role(r.oid, 'trenova_tenant', 'MEMBER')
            AND NOT pg_has_role(r.oid, 'trenova_rls_bypass', 'MEMBER')
    LOOP
        EXECUTE format('GRANT trenova_rls_bypass TO %I', existing.rolname);
    END LOOP;
END
$$;

--bun:split
CREATE SCHEMA IF NOT EXISTS trenova_rls;

--bun:split
REVOKE ALL ON SCHEMA trenova_rls FROM PUBLIC;

--bun:split
GRANT USAGE ON SCHEMA trenova_rls TO trenova_tenant, trenova_rls_bypass;

--bun:split
CREATE TABLE IF NOT EXISTS trenova_rls.scope_keys(
    "key_id" varchar(32) NOT NULL,
    "inner_pad" bytea NOT NULL,
    "outer_pad" bytea NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "retired_at" bigint,
    CONSTRAINT "pk_scope_keys" PRIMARY KEY ("key_id"),
    CONSTRAINT "ck_scope_keys_key_id" CHECK ("key_id" ~ '^[A-Za-z0-9_-]{1,32}$'),
    CONSTRAINT "ck_scope_keys_inner_pad" CHECK (octet_length("inner_pad") = 64),
    CONSTRAINT "ck_scope_keys_outer_pad" CHECK (octet_length("outer_pad") = 64)
);

--bun:split
REVOKE ALL ON trenova_rls.scope_keys FROM PUBLIC;

--bun:split
COMMENT ON TABLE trenova_rls.scope_keys IS 'HMAC-SHA256 pads for the keys that sign transaction tenant scopes; readable only by the schema owner through trenova_rls.claims()';

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.claims()
    RETURNS text[]
    LANGUAGE plpgsql
    STABLE
    SECURITY DEFINER
    PARALLEL RESTRICTED
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    token text := current_setting('trenova.scope', true);
    parts text[];
    key_row record;
BEGIN
    IF token IS NULL OR token = '' THEN
        RAISE EXCEPTION 'no tenant scope is set for this transaction'
            USING ERRCODE = '42501',
                  HINT = 'Run the statement inside a tenant-scoped transaction';
    END IF;

    parts := string_to_array(token, '.');
    IF cardinality(parts) <> 7 OR parts[1] <> 'v1' THEN
        RAISE EXCEPTION 'tenant scope is malformed' USING ERRCODE = '42501';
    END IF;

    SELECT k.inner_pad, k.outer_pad
    INTO key_row
    FROM trenova_rls.scope_keys k
    WHERE k.key_id = parts[2]
        AND k.retired_at IS NULL;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'tenant scope was signed with an unknown or retired key' USING ERRCODE = '42501';
    END IF;

    IF encode(
        sha256(key_row.outer_pad || sha256(key_row.inner_pad || convert_to(array_to_string(parts[1:6], '.'), 'UTF8'))),
        'hex'
    ) <> parts[7] THEN
        RAISE EXCEPTION 'tenant scope signature does not verify' USING ERRCODE = '42501';
    END IF;

    IF parts[6] !~ '^[0-9]{1,12}$' OR parts[6]::bigint < EXTRACT(EPOCH FROM transaction_timestamp())::bigint THEN
        RAISE EXCEPTION 'tenant scope has expired' USING ERRCODE = '42501';
    END IF;

    RETURN parts;
END
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.org_id()
    RETURNS text
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT (trenova_rls.claims())[3]
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.bu_id()
    RETURNS text
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT (trenova_rls.claims())[4]
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.user_id()
    RETURNS text
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT NULLIF((trenova_rls.claims())[5], '-')
$$;

--bun:split
REVOKE ALL ON FUNCTION trenova_rls.claims() FROM PUBLIC;

--bun:split
GRANT EXECUTE ON FUNCTION trenova_rls.claims(), trenova_rls.org_id(), trenova_rls.bu_id(), trenova_rls.user_id() TO trenova_tenant, trenova_rls_bypass;

--bun:split
CREATE TABLE IF NOT EXISTS trenova_rls.global_tables(
    "table_name" name NOT NULL,
    "reason" text NOT NULL,
    CONSTRAINT "pk_global_tables" PRIMARY KEY ("table_name"),
    CONSTRAINT "ck_global_tables_reason" CHECK (length(btrim("reason")) > 0)
);

--bun:split
REVOKE ALL ON trenova_rls.global_tables FROM PUBLIC;

--bun:split
GRANT SELECT ON trenova_rls.global_tables TO trenova_tenant, trenova_rls_bypass;

--bun:split
COMMENT ON TABLE trenova_rls.global_tables IS 'Tables in public that hold no tenant data and are therefore exempt from row-level security; every other table must carry a tenant policy';

--bun:split
INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES
    ('ai_audit_projector_state', 'Projector cursor shared by the audit worker across all tenants'),
    ('ai_retraining_cycles', 'Operator-level retraining cycles built from consenting organizations'),
    ('ai_training_exports', 'Operator-level training exports spanning consenting organizations'),
    ('bun_migration_locks', 'Migration bookkeeping'),
    ('bun_migrations', 'Migration bookkeeping'),
    ('business_units', 'Top of the tenant hierarchy; every tenant row references it'),
    ('dot_hazmat_references', 'Federal hazardous materials reference data'),
    ('edi_code_list_definitions', 'X12 code list definitions shared by every organization'),
    ('edi_document_types', 'X12 document type catalog shared by every organization'),
    ('edi_partner_setting_fields', 'Partner setting field catalog shared by every organization'),
    ('edi_source_context_fields', 'Source context field catalog shared by every organization'),
    ('edi_transaction_element_definitions', 'X12 element definitions shared by every organization'),
    ('edi_transaction_loop_definitions', 'X12 loop definitions shared by every organization'),
    ('edi_transaction_segment_definitions', 'X12 segment definitions shared by every organization'),
    ('edi_transaction_sets', 'X12 transaction set catalog shared by every organization'),
    ('hazmat_expirations', 'Federal hazardous materials expiration reference data'),
    ('ifta_jurisdictions', 'IFTA jurisdiction reference data maintained by the platform operator'),
    ('ifta_tax_rates', 'IFTA tax rates maintained by the platform operator'),
    ('jurisdiction_escort_thresholds', 'Oversize escort thresholds maintained by the platform operator'),
    ('jurisdiction_rules', 'Jurisdiction rules maintained by the platform operator'),
    ('pretrained_models', 'Model catalog shared by every organization'),
    ('seed_created_entities', 'Seeder bookkeeping'),
    ('spatial_ref_sys', 'PostGIS spatial reference catalog'),
    ('us_states', 'State reference data shared by every organization')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.apply_policy(target regclass, read_expr text, write_expr text DEFAULT NULL)
    RETURNS void
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    write_check text := COALESCE(write_expr, read_expr);
    existing record;
BEGIN
    EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', target);
    EXECUTE format('ALTER TABLE %s FORCE ROW LEVEL SECURITY', target);

    FOR existing IN
        SELECT p.polname
        FROM pg_policy p
        WHERE p.polrelid = target
            AND p.polname IN ('tenant_isolation', 'tenant_read', 'tenant_insert', 'tenant_update', 'tenant_delete', 'rls_bypass')
    LOOP
        EXECUTE format('DROP POLICY %I ON %s', existing.polname, target);
    END LOOP;

    IF write_check = read_expr THEN
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %s AS PERMISSIVE FOR ALL TO trenova_tenant USING (%s) WITH CHECK (%s)',
            target, read_expr, write_check
        );
    ELSE
        EXECUTE format('CREATE POLICY tenant_read ON %s AS PERMISSIVE FOR SELECT TO trenova_tenant USING (%s)', target, read_expr);
        EXECUTE format('CREATE POLICY tenant_insert ON %s AS PERMISSIVE FOR INSERT TO trenova_tenant WITH CHECK (%s)', target, write_check);
        EXECUTE format(
            'CREATE POLICY tenant_update ON %s AS PERMISSIVE FOR UPDATE TO trenova_tenant USING (%s) WITH CHECK (%s)',
            target, write_check, write_check
        );
        EXECUTE format('CREATE POLICY tenant_delete ON %s AS PERMISSIVE FOR DELETE TO trenova_tenant USING (%s)', target, write_check);
    END IF;

    EXECUTE format(
        'CREATE POLICY rls_bypass ON %s AS PERMISSIVE FOR ALL TO trenova_rls_bypass USING (true) WITH CHECK (true)',
        target
    );
END
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.standard_expr(org_column text DEFAULT 'organization_id', bu_column text DEFAULT 'business_unit_id', bu_nullable boolean DEFAULT false)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT format('%I = (SELECT trenova_rls.org_id())', org_column)
        || CASE
            WHEN bu_column IS NULL THEN ''
            WHEN bu_nullable THEN format(' AND (%1$I IS NULL OR %1$I = (SELECT trenova_rls.bu_id()))', bu_column)
            ELSE format(' AND %I = (SELECT trenova_rls.bu_id())', bu_column)
        END
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.reconcile()
    RETURNS integer
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    candidate record;
    applied integer := 0;
BEGIN
    EXECUTE 'GRANT USAGE ON SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    EXECUTE 'GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    EXECUTE 'REVOKE INSERT, UPDATE, DELETE ON public.bun_migrations, public.bun_migration_locks, public.seed_created_entities FROM trenova_tenant';

    FOR candidate IN
        SELECT c.oid::regclass AS target, bu.attnotnull AS bu_not_null
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute org ON org.attrelid = c.oid AND org.attname = 'organization_id' AND NOT org.attisdropped
        JOIN pg_attribute bu ON bu.attrelid = c.oid AND bu.attname = 'business_unit_id' AND NOT bu.attisdropped
        WHERE n.nspname = 'public'
            AND c.relkind IN ('r', 'p')
            AND org.attnotnull
            AND NOT c.relrowsecurity
            AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
    LOOP
        PERFORM trenova_rls.apply_policy(
            candidate.target,
            trenova_rls.standard_expr('organization_id', 'business_unit_id', NOT candidate.bu_not_null)
        );
        applied := applied + 1;
    END LOOP;

    RETURN applied;
END
$$;

--bun:split
REVOKE ALL ON FUNCTION trenova_rls.apply_policy(regclass, text, text), trenova_rls.reconcile() FROM PUBLIC;

--bun:split
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO trenova_tenant, trenova_rls_bypass;

--bun:split
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO trenova_tenant, trenova_rls_bypass;

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_partner_setting_schemas',
    'organization_id IS NULL OR (' || trenova_rls.standard_expr('organization_id', 'business_unit_id', true) || ')',
    trenova_rls.standard_expr('organization_id', 'business_unit_id', true)
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_source_context_schemas',
    'organization_id IS NULL OR (' || trenova_rls.standard_expr('organization_id', 'business_unit_id', true) || ')',
    trenova_rls.standard_expr('organization_id', 'business_unit_id', true)
);

--bun:split
SELECT trenova_rls.apply_policy('public.auth_events', trenova_rls.standard_expr('organization_id', 'business_unit_id', true));

--bun:split
SELECT trenova_rls.apply_policy('public.capture_pairings', trenova_rls.standard_expr('organization_id', 'business_unit_id', true));

--bun:split
SELECT trenova_rls.apply_policy('public.risk_decisions', trenova_rls.standard_expr('organization_id', 'business_unit_id', true));

--bun:split
SELECT trenova_rls.apply_policy('public.formula_schemas', trenova_rls.standard_expr('organization_id', NULL));

--bun:split
SELECT trenova_rls.apply_policy('public.provisioning_audit_records', trenova_rls.standard_expr('organization_id', NULL));

--bun:split
SELECT trenova_rls.apply_policy('public.scim_tokens', trenova_rls.standard_expr('organization_id', NULL));

--bun:split
SELECT trenova_rls.apply_policy('public.mfa_authenticators', trenova_rls.standard_expr('organization_id', NULL));

--bun:split
SELECT trenova_rls.apply_policy('public.user_role_assignments', trenova_rls.standard_expr('organization_id', NULL));

--bun:split
SELECT trenova_rls.apply_policy(
    'public.resource_permissions',
    $expr$EXISTS (
        SELECT 1
        FROM public.roles r
        WHERE r.id = resource_permissions.role_id
            AND r.organization_id = (SELECT trenova_rls.org_id())
            AND r.business_unit_id = (SELECT trenova_rls.bu_id())
    )$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.user_organization_memberships',
    '(' || trenova_rls.standard_expr() || ') OR user_id = (SELECT trenova_rls.user_id())',
    trenova_rls.standard_expr()
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.users',
    $expr$id = (SELECT trenova_rls.user_id())
        OR (current_organization_id = (SELECT trenova_rls.org_id()) AND business_unit_id = (SELECT trenova_rls.bu_id()))
        OR EXISTS (
            SELECT 1
            FROM public.user_organization_memberships m
            WHERE m.user_id = users.id
                AND m.organization_id = (SELECT trenova_rls.org_id())
                AND m.business_unit_id = (SELECT trenova_rls.bu_id())
        )$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.organizations',
    $expr$(id = (SELECT trenova_rls.org_id()) AND business_unit_id = (SELECT trenova_rls.bu_id()))
        OR EXISTS (
            SELECT 1
            FROM public.user_organization_memberships m
            WHERE m.organization_id = organizations.id
                AND m.user_id = (SELECT trenova_rls.user_id())
        )
        OR (
            business_unit_id = (SELECT trenova_rls.bu_id())
            AND (
                EXISTS (
                    SELECT 1
                    FROM public.edi_connections c
                    WHERE organizations.id IN (c.source_organization_id, c.target_organization_id)
                )
                OR EXISTS (
                    SELECT 1
                    FROM public.edi_load_tender_transfers t
                    WHERE organizations.id IN (t.source_organization_id, t.target_organization_id)
                )
            )
        )$expr$,
    $expr$id = (SELECT trenova_rls.org_id()) AND business_unit_id = (SELECT trenova_rls.bu_id())$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_connections',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())
        AND (SELECT trenova_rls.org_id()) IN (source_organization_id, target_organization_id)$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_shipment_links',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())
        AND (SELECT trenova_rls.org_id()) IN (source_organization_id, target_organization_id)$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_load_tender_transfers',
    $expr$(source_organization_id = (SELECT trenova_rls.org_id()) AND source_business_unit_id = (SELECT trenova_rls.bu_id()))
        OR (target_organization_id = (SELECT trenova_rls.org_id()) AND target_business_unit_id = (SELECT trenova_rls.bu_id()))$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_tender_recipients',
    $expr$(source_organization_id = (SELECT trenova_rls.org_id()) AND source_business_unit_id = (SELECT trenova_rls.bu_id()))
        OR (
            recipient_organization_id = (SELECT trenova_rls.org_id())
            AND (
                recipient_business_unit_id = (SELECT trenova_rls.bu_id())
                OR recipient_business_unit_id IS NULL
                OR business_unit_id = (SELECT trenova_rls.bu_id())
            )
        )$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_tender_changes',
    $expr$(source_organization_id = (SELECT trenova_rls.org_id()) AND source_business_unit_id = (SELECT trenova_rls.bu_id()))
        OR EXISTS (
            SELECT 1
            FROM public.edi_tender_recipients r
            WHERE r.id = edi_tender_changes.recipient_id
                AND r.recipient_organization_id = (SELECT trenova_rls.org_id())
        )$expr$
);

--bun:split
SELECT trenova_rls.apply_policy(
    'public.edi_transfer_changes',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())
        AND EXISTS (
            SELECT 1
            FROM public.edi_shipment_links l
            WHERE l.id = edi_transfer_changes.shipment_link_id
        )$expr$
);

--bun:split
SELECT trenova_rls.reconcile();
