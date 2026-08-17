-- Finds shipments whose stored travel envelope disagrees with the envelope its
-- own cargo lines imply.
--
-- Context: updateCommodity omitted length_feet, width_feet and height_feet from
-- its SET list, so per-line dimension edits never persisted while the envelope
-- was still recomputed from the submitted values and saved. Any shipment saved
-- with edited line dimensions between the dimensions feature landing and the
-- fix can hold an envelope its stored lines do not support. Overall height
-- drives permit thresholds, so a stale envelope can overstate clearance.
--
-- Mirrors ComputeEnvelope/ResolvedXFeet/ApplyEnvelope in
-- internal/core/domain/shipment/envelope.go:
--   * each dimension resolves line value, then commodity default, then zero
--   * envelope takes the maximum across lines, length included
--   * overall height adds deck height only when cargo height is non-zero
--   * an all-zero envelope stores NULL in all four columns
--   * a zero component stores NULL (nonZero)
-- Deck height comes from the trailer type, in inches, and is zero when the
-- shipment has no trailer type.
--
-- Read-only. Comparison rounds to 2dp to match NUMERIC(10,2) storage.
--
-- On remediation: the dropped line dimensions were never written, so deploying
-- the fix does not bring them back. Re-saving an affected shipment recomputes
-- the envelope from the stored lines and clears it. Note though that on a
-- shipment with a single cargo line the stored envelope IS that line's lost
-- dimensions, since the envelope is the maximum over one line, minus the deck
-- height on overall. Those rows can be repaired from this query's stored_*
-- columns before anything overwrites them. On multi-line shipments only the
-- per-dimension maximum survives, so the individual lines cannot be recovered
-- and need re-entry.

WITH resolved AS (
    SELECT
        sc.shipment_id,
        MAX(COALESCE(sc.length_feet, com.length_per_unit, 0)) AS length_feet,
        MAX(COALESCE(sc.width_feet,  com.width_per_unit,  0)) AS width_feet,
        MAX(COALESCE(sc.height_feet, com.height_per_unit, 0)) AS height_feet
    FROM shipment_commodities AS sc
    LEFT JOIN commodities AS com
           ON com.id               = sc.commodity_id
          AND com.organization_id  = sc.organization_id
          AND com.business_unit_id = sc.business_unit_id
    GROUP BY sc.shipment_id
),
expected AS (
    SELECT
        sp.id,
        sp.pro_number,
        sp.status,
        sp.organization_id,
        sp.business_unit_id,
        sp.envelope_length_feet         AS stored_length,
        sp.envelope_width_feet          AS stored_width,
        sp.envelope_height_feet         AS stored_height,
        sp.envelope_overall_height_feet AS stored_overall,
        CASE WHEN COALESCE(r.length_feet, 0) = 0 AND COALESCE(r.width_feet, 0) = 0
                  AND COALESCE(r.height_feet, 0) = 0
             THEN NULL ELSE NULLIF(r.length_feet, 0) END AS expected_length,
        CASE WHEN COALESCE(r.length_feet, 0) = 0 AND COALESCE(r.width_feet, 0) = 0
                  AND COALESCE(r.height_feet, 0) = 0
             THEN NULL ELSE NULLIF(r.width_feet, 0) END  AS expected_width,
        CASE WHEN COALESCE(r.length_feet, 0) = 0 AND COALESCE(r.width_feet, 0) = 0
                  AND COALESCE(r.height_feet, 0) = 0
             THEN NULL ELSE NULLIF(r.height_feet, 0) END AS expected_height,
        CASE WHEN COALESCE(r.length_feet, 0) = 0 AND COALESCE(r.width_feet, 0) = 0
                  AND COALESCE(r.height_feet, 0) = 0
             THEN NULL
             WHEN COALESCE(r.height_feet, 0) > 0
             THEN NULLIF(r.height_feet + COALESCE(et.deck_height_inches, 0) / 12.0, 0)
             ELSE NULL END AS expected_overall
    FROM shipments AS sp
    LEFT JOIN resolved AS r ON r.shipment_id = sp.id
    LEFT JOIN equipment_types AS et
           ON et.id               = sp.trailer_type_id
          AND et.organization_id  = sp.organization_id
          AND et.business_unit_id = sp.business_unit_id
)
SELECT
    id,
    pro_number,
    status,
    organization_id,
    business_unit_id,
    stored_length,  ROUND(expected_length,  2) AS expected_length,
    stored_width,   ROUND(expected_width,   2) AS expected_width,
    stored_height,  ROUND(expected_height,  2) AS expected_height,
    stored_overall, ROUND(expected_overall, 2) AS expected_overall,
    CASE
        WHEN stored_overall IS NOT NULL
         AND (expected_overall IS NULL OR stored_overall > ROUND(expected_overall, 2))
        THEN 'overstates clearance'
        ELSE 'understates or differs'
    END AS drift
FROM expected
WHERE ROUND(stored_length,  2) IS DISTINCT FROM ROUND(expected_length,  2)
   OR ROUND(stored_width,   2) IS DISTINCT FROM ROUND(expected_width,   2)
   OR ROUND(stored_height,  2) IS DISTINCT FROM ROUND(expected_height,  2)
   OR ROUND(stored_overall, 2) IS DISTINCT FROM ROUND(expected_overall, 2)
ORDER BY
    CASE WHEN stored_overall IS NOT NULL
          AND (expected_overall IS NULL OR stored_overall > ROUND(expected_overall, 2))
         THEN 0 ELSE 1 END,
    pro_number;
