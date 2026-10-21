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
