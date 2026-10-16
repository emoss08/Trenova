-- A role's "own" or "team" data scope is enforced for a record kind by a
-- restrictive policy that reads the request's read scope, which the
-- application sends with each transaction as trenova.read_scope:
-- "<kind>:<scope>[:<id>,<id>]" entries joined by ";". The value only ever
-- narrows what the signed tenant scope already allows, so it needs no
-- signature: a forged or missing one can at most widen a request back to its
-- own organization.
CREATE OR REPLACE FUNCTION trenova_rls.read_scope(kind text)
    RETURNS text
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT COALESCE((
        SELECT split_part(entry, ':', 2)
        FROM unnest(string_to_array(NULLIF(current_setting('trenova.read_scope', true), ''), ';')) AS entry
        WHERE split_part(entry, ':', 1) = kind
        LIMIT 1
    ), '')
$$;

--bun:split
CREATE OR REPLACE FUNCTION trenova_rls.read_scope_ids(kind text)
    RETURNS text[]
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT COALESCE((
        SELECT string_to_array(NULLIF(split_part(entry, ':', 3), ''), ',')
        FROM unnest(string_to_array(NULLIF(current_setting('trenova.read_scope', true), ''), ';')) AS entry
        WHERE split_part(entry, ':', 1) = kind
        LIMIT 1
    ), ARRAY[]::text[])
$$;

--bun:split
DROP POLICY IF EXISTS read_scope ON public.shipments;

--bun:split
CREATE POLICY read_scope ON public.shipments AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR owner_id = (SELECT trenova_rls.user_id())
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.orders;

--bun:split
CREATE POLICY read_scope ON public.orders AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('order')) <> 'own'
        OR owner_id = (SELECT trenova_rls.user_id())
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.invoices;

--bun:split
CREATE POLICY read_scope ON public.invoices AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('invoice')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = invoices.shipment_id
                AND s.organization_id = invoices.organization_id
                AND s.business_unit_id = invoices.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
        OR EXISTS (
            SELECT 1
            FROM public.orders o
            WHERE o.id = invoices.order_id
                AND o.organization_id = invoices.organization_id
                AND o.business_unit_id = invoices.business_unit_id
                AND o.owner_id = (SELECT trenova_rls.user_id())
        )
        OR EXISTS (
            SELECT 1
            FROM public.invoice_lines l
            JOIN public.shipments s
                ON s.id = l.shipment_id
                AND s.organization_id = l.organization_id
                AND s.business_unit_id = l.business_unit_id
            WHERE l.invoice_id = invoices.id
                AND l.organization_id = invoices.organization_id
                AND l.business_unit_id = invoices.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.workers;

--bun:split
CREATE POLICY read_scope ON public.workers AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) <> 'team'
        OR manager_id = ANY ((SELECT trenova_rls.read_scope_ids('worker'))::text[])
        OR EXISTS (
            SELECT 1
            FROM public.fleet_codes fc
            WHERE fc.id = workers.fleet_code_id
                AND fc.organization_id = workers.organization_id
                AND fc.business_unit_id = workers.business_unit_id
                AND fc.manager_id = ANY ((SELECT trenova_rls.read_scope_ids('worker'))::text[])
        )
    );

--bun:split
-- An "own" or "team" grant on a kind that enforces neither was always read
-- as organization-wide. It is stored as what it was, so the role editor and
-- the audit trail no longer claim a limit nothing applied.
UPDATE "resource_permissions"
SET "data_scope" = 'organization',
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE ("data_scope" = 'own' AND "resource" NOT IN ('shipment', 'order', 'invoice', 'capture_batch', 'report'))
    OR ("data_scope" = 'team' AND "resource" NOT IN ('worker', 'worker_schedule', 'timesheet', 'profile_change_request'));

--bun:split
UPDATE "api_key_permissions"
SET "data_scope" = 'organization',
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE ("data_scope" = 'own' AND "resource" NOT IN ('shipment', 'order', 'invoice', 'capture_batch', 'report'))
    OR ("data_scope" = 'team' AND "resource" NOT IN ('worker', 'worker_schedule', 'timesheet', 'profile_change_request'));
