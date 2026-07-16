--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- One-shot repair of order derived state. The original backfill hardcoded every
-- order to 'Confirmed' and summed canceled legs into total_amount, and until the
-- derivation hooks landed most leg transitions never re-derived the parent order.
-- This mirrors order.Derive (internal/core/domain/order/status.go): canceled legs
-- are excluded; all-invoiced → Billed; all-delivered → Completed; all-new →
-- Confirmed; otherwise InProgress; all-canceled → Canceled; no legs → Draft.
-- Closed is terminal and is never overwritten.
WITH leg_stats AS (
    SELECT
        s.order_id,
        s.organization_id,
        s.business_unit_id,
        COUNT(*) AS total_legs,
        COUNT(*) FILTER (WHERE s.status != 'Canceled') AS active_legs,
        COUNT(*) FILTER (WHERE s.status = 'Invoiced') AS invoiced_legs,
        COUNT(*) FILTER (WHERE s.status IN ('ReadyToInvoice', 'Completed', 'Invoiced')) AS delivered_legs,
        COUNT(*) FILTER (WHERE s.status = 'New') AS new_legs
    FROM
        shipments s
    WHERE
        s.order_id IS NOT NULL
    GROUP BY
        s.order_id,
        s.organization_id,
        s.business_unit_id
)
UPDATE
    orders o
SET
    status = CASE WHEN ls.active_legs = 0 THEN
        'Canceled'
    WHEN ls.invoiced_legs = ls.active_legs THEN
        'Billed'
    WHEN ls.delivered_legs = ls.active_legs THEN
        'Completed'
    WHEN ls.new_legs = ls.active_legs THEN
        'Confirmed'
    ELSE
        'InProgress'
    END::order_status_enum,
    version = o.version + 1,
    updated_at = EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM
    leg_stats ls
WHERE
    o.id = ls.order_id
    AND o.organization_id = ls.organization_id
    AND o.business_unit_id = ls.business_unit_id
    AND o.status != 'Closed'
    AND o.status != CASE WHEN ls.active_legs = 0 THEN
        'Canceled'
    WHEN ls.invoiced_legs = ls.active_legs THEN
        'Billed'
    WHEN ls.delivered_legs = ls.active_legs THEN
        'Completed'
    WHEN ls.new_legs = ls.active_legs THEN
        'Confirmed'
    ELSE
        'InProgress'
    END::order_status_enum;

--bun:split
-- Recompute every order's AR total: non-canceled leg charges + order-level charges.
WITH totals AS (
    SELECT
        o.id,
        o.organization_id,
        o.business_unit_id,
        COALESCE((
            SELECT
                SUM(s.total_charge_amount)
            FROM shipments s
            WHERE
                s.order_id = o.id
                AND s.organization_id = o.organization_id
                AND s.business_unit_id = o.business_unit_id
                AND s.status != 'Canceled'), 0) + COALESCE((
            SELECT
                SUM(oc.amount)
            FROM order_charges oc
            WHERE
                oc.order_id = o.id
                AND oc.organization_id = o.organization_id
                AND oc.business_unit_id = o.business_unit_id), 0) AS new_total
    FROM
        orders o
)
UPDATE
    orders o
SET
    total_amount = t.new_total,
    version = o.version + 1,
    updated_at = EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM
    totals t
WHERE
    o.id = t.id
    AND o.organization_id = t.organization_id
    AND o.business_unit_id = t.business_unit_id
    AND o.total_amount IS DISTINCT FROM t.new_total;
