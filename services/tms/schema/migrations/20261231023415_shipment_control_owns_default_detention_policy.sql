-- The organization's default detention policy has one owner: shipment control's
-- default_detention_policy_id, which the detention engine reads as the fallback for a stop no
-- other policy covers. The per-policy flag that said the same thing a second way is dropped.
-- An organization that marked a policy as its default keeps it, now on shipment control.
UPDATE "shipment_controls" AS sc
SET "default_detention_policy_id" = dp."id"
FROM "detention_policies" AS dp
WHERE dp."organization_id" = sc."organization_id"
    AND dp."business_unit_id" = sc."business_unit_id"
    AND dp."is_org_default"
    AND sc."default_detention_policy_id" IS NULL;

DROP INDEX IF EXISTS "uq_detention_policies_org_default";

ALTER TABLE "detention_policies"
    DROP CONSTRAINT IF EXISTS "chk_detention_policies_org_default_scope";

ALTER TABLE "detention_policies"
    DROP COLUMN IF EXISTS "is_org_default";

-- A policy that is the organization's default cannot be deleted out from under the engine. The
-- check runs at the end of the statement, so removing a whole organization, which cascades to
-- both tables, still works.
ALTER TABLE "shipment_controls"
    DROP CONSTRAINT IF EXISTS "fk_shipment_controls_default_detention_policy";

ALTER TABLE "shipment_controls"
    ADD CONSTRAINT "fk_shipment_controls_default_detention_policy" FOREIGN KEY ("default_detention_policy_id", "organization_id", "business_unit_id") REFERENCES "detention_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE NO ACTION;

COMMENT ON COLUMN "shipment_controls"."default_detention_policy_id" IS 'The detention policy that governs a stop no other active policy covers; required while the policy engine is on';
