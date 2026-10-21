-- A principal that cannot read a kind at all is sent "<kind>:none". Until now
-- such a kind was left out of trenova.read_scope, which reads as the whole
-- organization, so a relation, loader or join reached records the person may
-- not read. Under "none" a person still sees what is their own: the worker
-- linked to their user, and the shipments they are assigned to drive.
DROP POLICY IF EXISTS read_scope ON public.shipments;

--bun:split
CREATE POLICY read_scope ON public.shipments AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) NOT IN ('own', 'none')
        OR (
            (SELECT trenova_rls.read_scope('shipment')) = 'own'
            AND owner_id = (SELECT trenova_rls.user_id())
        )
        OR (
            (SELECT trenova_rls.read_scope('shipment')) = 'none'
            AND EXISTS (
                SELECT 1
                FROM public.shipment_moves sm
                JOIN public.assignments a
                    ON a.shipment_move_id = sm.id
                    AND a.organization_id = sm.organization_id
                    AND a.business_unit_id = sm.business_unit_id
                JOIN public.workers w
                    ON w.id IN (a.primary_worker_id, a.secondary_worker_id)
                    AND w.organization_id = a.organization_id
                    AND w.business_unit_id = a.business_unit_id
                WHERE sm.shipment_id = shipments.id
                    AND sm.organization_id = shipments.organization_id
                    AND sm.business_unit_id = shipments.business_unit_id
                    AND w.user_id = (SELECT trenova_rls.user_id())
            )
        )
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.orders;

--bun:split
CREATE POLICY read_scope ON public.orders AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('order')) NOT IN ('own', 'none')
        OR (
            (SELECT trenova_rls.read_scope('order')) = 'own'
            AND owner_id = (SELECT trenova_rls.user_id())
        )
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.invoices;

--bun:split
CREATE POLICY read_scope ON public.invoices AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('invoice')) NOT IN ('own', 'none')
        OR (
            (SELECT trenova_rls.read_scope('invoice')) = 'own'
            AND (
                EXISTS (
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
            )
        )
    );

--bun:split
DROP POLICY IF EXISTS read_scope ON public.workers;

--bun:split
CREATE POLICY read_scope ON public.workers AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) NOT IN ('team', 'none')
        OR (
            (SELECT trenova_rls.read_scope('worker')) = 'team'
            AND (
                manager_id = ANY ((SELECT trenova_rls.read_scope_ids('worker'))::text[])
                OR EXISTS (
                    SELECT 1
                    FROM public.fleet_codes fc
                    WHERE fc.id = workers.fleet_code_id
                        AND fc.organization_id = workers.organization_id
                        AND fc.business_unit_id = workers.business_unit_id
                        AND fc.manager_id = ANY ((SELECT trenova_rls.read_scope_ids('worker'))::text[])
                )
            )
        )
        OR (
            (SELECT trenova_rls.read_scope('worker')) = 'none'
            AND user_id = (SELECT trenova_rls.user_id())
        )
    );
