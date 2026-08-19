-- Rate tables merge into rate matrices.
--
-- A rate table was always a rate matrix with one axis: an exact table is an
-- Exact-mode axis keyed by string, a range table is a Range-mode axis banded
-- by number. Two concepts for one shape meant two editors, two APIs and two
-- answers to "where do I put this tariff". The storage converges here; the
-- formula language keeps its lookup("key", x) vocabulary, now resolved against
-- single-axis matrices by code.
--
-- The identifier payload of each PULID is reused under the new prefix, so the
-- migrated rows keep their creation ordering and the same table always
-- becomes the same matrix.
DO $$
DECLARE
    conflicted text;
BEGIN
    SELECT string_agg(DISTINCT rt.key, ', ')
      INTO conflicted
      FROM rate_tables rt
      JOIN rate_matrices rm
        ON rm.organization_id = rt.organization_id
       AND rm.business_unit_id = rt.business_unit_id
       AND lower(rm.code) = lower(rt.key)
     WHERE 'rmx_' || substr(rt.id, 4) <> rm.id;
    IF conflicted IS NOT NULL THEN
        RAISE EXCEPTION
            'rate table keys collide with existing rate matrix codes: %. Rename one side before migrating; a silent rename would break every formula that reads lookup() by that name.',
            conflicted;
    END IF;
END
$$;

--bun:split
INSERT INTO "rate_matrices"(
    "id", "business_unit_id", "organization_id",
    "code", "name", "description",
    "currency", "status",
    "rounding_mode", "rounding_precision",
    "version", "created_at", "updated_at"
)
SELECT
    'rmx_' || substr(rt.id, 4),
    rt.business_unit_id,
    rt.organization_id,
    rt.key,
    rt.name,
    rt.description,
    'USD',
    CASE WHEN rt.active THEN 'Active'::status_enum ELSE 'Inactive'::status_enum END,
    'HalfUp',
    2,
    rt.version,
    rt.created_at,
    rt.updated_at
FROM "rate_tables" rt
ON CONFLICT DO NOTHING;

--bun:split
INSERT INTO "rate_matrix_dimensions"(
    "id", "business_unit_id", "organization_id", "rate_matrix_id",
    "position", "kind", "match_mode", "label",
    "created_at", "updated_at"
)
SELECT
    'rmd_' || substr(rt.id, 4),
    rt.business_unit_id,
    rt.organization_id,
    'rmx_' || substr(rt.id, 4),
    0,
    CASE WHEN rt.lookup_type = 'Range' THEN 'Quantity'::rate_matrix_dimension_kind_enum
         ELSE 'Custom'::rate_matrix_dimension_kind_enum END,
    CASE WHEN rt.lookup_type = 'Range' THEN 'Range'::rate_matrix_match_mode_enum
         ELSE 'Exact'::rate_matrix_match_mode_enum END,
    rt.name,
    rt.created_at,
    rt.updated_at
FROM "rate_tables" rt
ON CONFLICT DO NOTHING;

--bun:split
-- Entries the old lookup silently skipped — an exact entry with no key, a
-- range entry with no floor — are skipped here too. They never produced a
-- rate, and copying them would either corrupt the axis or trip the cell
-- constraints for rows that were already dead.
INSERT INTO "rate_matrix_cells"(
    "id", "business_unit_id", "organization_id", "rate_matrix_id",
    "d0_key", "d0_min", "d0_max",
    "value", "deficit_eligible",
    "created_at", "updated_at"
)
SELECT
    'rmc_' || substr(rte.id, 5),
    rte.business_unit_id,
    rte.organization_id,
    'rmx_' || substr(rte.rate_table_id, 4),
    CASE WHEN rt.lookup_type = 'Exact' THEN rte.match_key ELSE '' END,
    CASE WHEN rt.lookup_type = 'Range' THEN rte.range_min ELSE NULL END,
    CASE WHEN rt.lookup_type = 'Range' THEN rte.range_max ELSE NULL END,
    rte.value,
    TRUE,
    rte.created_at,
    rte.updated_at
FROM "rate_table_entries" rte
JOIN "rate_tables" rt
  ON rt.id = rte.rate_table_id
 AND rt.organization_id = rte.organization_id
 AND rt.business_unit_id = rte.business_unit_id
WHERE (rt.lookup_type = 'Exact' AND rte.match_key IS NOT NULL AND rte.match_key <> '')
   OR (rt.lookup_type = 'Range' AND rte.range_min IS NOT NULL)
ON CONFLICT DO NOTHING;

--bun:split
DROP TABLE IF EXISTS "rate_table_entries";

--bun:split
DROP TABLE IF EXISTS "rate_tables";

--bun:split
DROP TYPE IF EXISTS "rate_table_lookup_type_enum";
