--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Cancelling a shipment used to skip its live assignments (the filter matched only
-- archived rows) and overwrite completed stops and moves with Canceled. Cancel now
-- cancels live assignments and leaves completed work alone; this brings the rows
-- already canceled the old way into the same shape.
UPDATE
    assignments a
SET
    status = 'Canceled',
    version = a.version + 1,
    updated_at = extract(epoch FROM current_timestamp)::bigint
FROM
    shipment_moves sm
    JOIN shipments s ON s.id = sm.shipment_id
        AND s.organization_id = sm.organization_id
        AND s.business_unit_id = sm.business_unit_id
WHERE
    a.shipment_move_id = sm.id
    AND a.organization_id = sm.organization_id
    AND a.business_unit_id = sm.business_unit_id
    AND s.status = 'Canceled'
    AND a.archived_at IS NULL
    AND a.status IN ('New', 'InProgress');

--bun:split
UPDATE
    stops st
SET
    status = 'Completed',
    version = st.version + 1,
    updated_at = extract(epoch FROM current_timestamp)::bigint
FROM
    shipment_moves sm
    JOIN shipments s ON s.id = sm.shipment_id
        AND s.organization_id = sm.organization_id
        AND s.business_unit_id = sm.business_unit_id
WHERE
    st.shipment_move_id = sm.id
    AND st.organization_id = sm.organization_id
    AND st.business_unit_id = sm.business_unit_id
    AND s.status = 'Canceled'
    AND st.status = 'Canceled'
    AND st.actual_arrival IS NOT NULL
    AND st.actual_departure IS NOT NULL;

--bun:split
UPDATE
    shipment_moves sm
SET
    status = 'Completed',
    version = sm.version + 1,
    updated_at = extract(epoch FROM current_timestamp)::bigint
FROM
    shipments s
WHERE
    s.id = sm.shipment_id
    AND s.organization_id = sm.organization_id
    AND s.business_unit_id = sm.business_unit_id
    AND s.status = 'Canceled'
    AND sm.status = 'Canceled'
    AND EXISTS (
        SELECT
            1
        FROM
            stops st
        WHERE
            st.shipment_move_id = sm.id
            AND st.organization_id = sm.organization_id
            AND st.business_unit_id = sm.business_unit_id
            AND st.status = 'Completed')
    AND NOT EXISTS (
        SELECT
            1
        FROM
            stops st
        WHERE
            st.shipment_move_id = sm.id
            AND st.organization_id = sm.organization_id
            AND st.business_unit_id = sm.business_unit_id
            AND st.status NOT IN ('Completed', 'Canceled'));
