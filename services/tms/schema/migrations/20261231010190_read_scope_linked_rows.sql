-- A principal that cannot read a kind is no longer sent "<kind>:none", so the
-- policies drop that branch. "none" hid every row of the kind from the
-- relations, loaders and joins that reach it from a record the person can
-- read, so under row-level security a tractor's driver or a shipment's order
-- came back null where the application would show its reference view. The
-- application now decides those linked rows (the GraphQL field guard, the
-- REST response guard, search, reports and analytics, all through refview),
-- and refuses a root read of a kind the person cannot read before it reaches
-- the database. The "own" and "team" scopes still narrow every read of a
-- person who can read the kind.
DROP POLICY IF EXISTS read_scope ON public.shipments;

CREATE POLICY read_scope ON public.shipments AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR owner_id = (SELECT trenova_rls.user_id())
    );

DROP POLICY IF EXISTS read_scope ON public.orders;

CREATE POLICY read_scope ON public.orders AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('order')) <> 'own'
        OR owner_id = (SELECT trenova_rls.user_id())
    );

DROP POLICY IF EXISTS read_scope ON public.invoices;

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

DROP POLICY IF EXISTS read_scope ON public.workers;

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
