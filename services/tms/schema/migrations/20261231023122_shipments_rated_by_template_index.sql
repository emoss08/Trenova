--keel:notx
-- A formula backtest reads the shipments a template last rated, newest first: this index
-- returns them in order instead of sorting every shipment the template ever rated. It leads
-- with the template like the index it replaces, so a lookup by template alone still uses it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_formula_template_created" ON "shipments"("formula_template_id", "organization_id", "business_unit_id", "created_at" DESC, "id" DESC)
WHERE
    "formula_template_id" IS NOT NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipments_formula_template";
