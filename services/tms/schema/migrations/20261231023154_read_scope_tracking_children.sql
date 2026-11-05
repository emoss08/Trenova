-- Own-scoped shipment readers could still read what these tables hold about everyone else's
-- shipments: each is queried directly, with an optional shipment id, or reached by a token. A row
-- that exists only as part of a shipment follows that shipment for every reader.
--
-- Left out on purpose: shipment_moves, stops and assignments are consulted by equipment and driver
-- conflict checks, where a hidden row reads as "nothing in the way"; invoices, service failures,
-- tenders, rate confirmations and detention occurrences are their own permission resources, read
-- across shipments by their own grants.
DROP POLICY IF EXISTS read_scope ON public.tracking_events;

CREATE POLICY read_scope ON public.tracking_events AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = tracking_events.shipment_id
                AND s.organization_id = tracking_events.organization_id
                AND s.business_unit_id = tracking_events.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.check_calls;

CREATE POLICY read_scope ON public.check_calls AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = check_calls.shipment_id
                AND s.organization_id = check_calls.organization_id
                AND s.business_unit_id = check_calls.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.carrier_tracking_links;

CREATE POLICY read_scope ON public.carrier_tracking_links AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = carrier_tracking_links.shipment_id
                AND s.organization_id = carrier_tracking_links.organization_id
                AND s.business_unit_id = carrier_tracking_links.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.customer_tracking_links;

CREATE POLICY read_scope ON public.customer_tracking_links AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = customer_tracking_links.shipment_id
                AND s.organization_id = customer_tracking_links.organization_id
                AND s.business_unit_id = customer_tracking_links.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.customer_status_notifications;

CREATE POLICY read_scope ON public.customer_status_notifications AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = customer_status_notifications.shipment_id
                AND s.organization_id = customer_status_notifications.organization_id
                AND s.business_unit_id = customer_status_notifications.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.stop_etas;

CREATE POLICY read_scope ON public.stop_etas AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = stop_etas.shipment_id
                AND s.organization_id = stop_etas.organization_id
                AND s.business_unit_id = stop_etas.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.shipment_holds;

CREATE POLICY read_scope ON public.shipment_holds AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_holds.shipment_id
                AND s.organization_id = shipment_holds.organization_id
                AND s.business_unit_id = shipment_holds.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.shipment_commodities;

CREATE POLICY read_scope ON public.shipment_commodities AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_commodities.shipment_id
                AND s.organization_id = shipment_commodities.organization_id
                AND s.business_unit_id = shipment_commodities.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.additional_charges;

CREATE POLICY read_scope ON public.additional_charges AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = additional_charges.shipment_id
                AND s.organization_id = additional_charges.organization_id
                AND s.business_unit_id = additional_charges.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.shipment_comment_mentions;

CREATE POLICY read_scope ON public.shipment_comment_mentions AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_comment_mentions.shipment_id
                AND s.organization_id = shipment_comment_mentions.organization_id
                AND s.business_unit_id = shipment_comment_mentions.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.shipment_comment_acknowledgments;

CREATE POLICY read_scope ON public.shipment_comment_acknowledgments AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = shipment_comment_acknowledgments.shipment_id
                AND s.organization_id = shipment_comment_acknowledgments.organization_id
                AND s.business_unit_id = shipment_comment_acknowledgments.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.permits;

CREATE POLICY read_scope ON public.permits AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = permits.shipment_id
                AND s.organization_id = permits.organization_id
                AND s.business_unit_id = permits.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );

DROP POLICY IF EXISTS read_scope ON public.permit_requirements;

CREATE POLICY read_scope ON public.permit_requirements AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('shipment')) <> 'own'
        OR EXISTS (
            SELECT 1
            FROM public.shipments s
            WHERE s.id = permit_requirements.shipment_id
                AND s.organization_id = permit_requirements.organization_id
                AND s.business_unit_id = permit_requirements.business_unit_id
                AND s.owner_id = (SELECT trenova_rls.user_id())
        )
    );
