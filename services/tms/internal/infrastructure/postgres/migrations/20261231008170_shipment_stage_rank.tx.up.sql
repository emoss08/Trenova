-- The board groups, sorts and filters shipments by stage. The stage is a pure
-- function of status (shipment.StageRankSQL), stored so it indexes and
-- paginates like any other column.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "stage_rank" smallint GENERATED ALWAYS AS (CASE WHEN "status" IN ('Delayed') THEN 1 WHEN "status" IN ('New', 'PartiallyAssigned') THEN 2 WHEN "status" IN ('InTransit', 'PartiallyCompleted') THEN 3 WHEN "status" IN ('Assigned') THEN 4 WHEN "status" IN ('Completed', 'ReadyToInvoice', 'Invoiced') THEN 5 WHEN "status" IN ('Canceled') THEN 6 ELSE 2 END) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_stage_rank" ON "shipments"("organization_id", "business_unit_id", "stage_rank", "created_at" DESC, "id" DESC);
