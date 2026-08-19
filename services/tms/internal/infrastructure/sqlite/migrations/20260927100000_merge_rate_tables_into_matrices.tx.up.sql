-- Hand-completed SQLite translation of the rate table merge; the converter
-- cannot carry the PostgreSQL version's procedural collision guard or its
-- enum casts, so this file no longer regenerates. See docs/databases.md.
-- Source: 20260927100000_merge_rate_tables_into_matrices.tx.up.sql
--
-- The collision guard is played here by uq_rate_matrices_code: a rate table
-- key already claimed by a matrix fails the insert and aborts the migration,
-- which is the same loud refusal the PostgreSQL DO block gives, with a less
-- helpful message. Entries the old lookup skipped (keyless exact rows,
-- floorless range rows) are skipped by the copy too.
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
    rt."key",
    rt.name,
    rt.description,
    'USD',
    CASE WHEN rt.active THEN 'Active' ELSE 'Inactive' END,
    'HalfUp',
    2,
    rt.version,
    rt.created_at,
    rt.updated_at
FROM "rate_tables" rt;

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
    CASE WHEN rt.lookup_type = 'Range' THEN 'Quantity' ELSE 'Custom' END,
    CASE WHEN rt.lookup_type = 'Range' THEN 'Range' ELSE 'Exact' END,
    rt.name,
    rt.created_at,
    rt.updated_at
FROM "rate_tables" rt;

--bun:split
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
    1,
    rte.created_at,
    rte.updated_at
FROM "rate_table_entries" rte
JOIN "rate_tables" rt
  ON rt.id = rte.rate_table_id
 AND rt.organization_id = rte.organization_id
 AND rt.business_unit_id = rte.business_unit_id
WHERE (rt.lookup_type = 'Exact' AND rte.match_key IS NOT NULL AND rte.match_key <> '')
   OR (rt.lookup_type = 'Range' AND rte.range_min IS NOT NULL);

--bun:split
DROP TABLE IF EXISTS "rate_table_entries";

--bun:split
DROP TABLE IF EXISTS "rate_tables";
