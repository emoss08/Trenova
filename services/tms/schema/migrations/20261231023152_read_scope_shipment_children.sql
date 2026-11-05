-- The "own" scope lived on shipments alone, so a reader limited to their own shipments could read
-- the events and comments of everyone else's: these tables are queried directly, with an optional
-- shipment id. A child row now follows the shipment it belongs to, for every reader.
DROP POLICY IF EXISTS read_scope ON public.shipment_events;

CREATE POLICY read_scope ON public.shipment_events AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_events.shipment_id
                AND s.organization_id = shipment_events.organization_id
                AND s.business_unit_id = shipment_events.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.shipment_comments;

CREATE POLICY read_scope ON public.shipment_comments AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_comments.shipment_id
                AND s.organization_id = shipment_comments.organization_id
                AND s.business_unit_id = shipment_comments.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );
