WITH ranked_assignments AS (
    SELECT
        id,
        organization_id,
        business_unit_id,
        ROW_NUMBER() OVER (
            PARTITION BY shipment_move_id, organization_id, business_unit_id
            ORDER BY updated_at DESC, created_at DESC, id DESC
        ) AS row_num
    FROM assignments
)
DELETE FROM assignments a
USING ranked_assignments ra
WHERE a.id = ra.id
  AND a.organization_id = ra.organization_id
  AND a.business_unit_id = ra.business_unit_id
  AND ra.row_num > 1;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assignments_move_tenant"
    ON "assignments"("shipment_move_id", "organization_id", "business_unit_id");
