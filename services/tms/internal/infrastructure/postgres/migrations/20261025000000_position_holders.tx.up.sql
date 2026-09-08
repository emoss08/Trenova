--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
--
-- Who holds a position. Driving positions are held by workers, who already
-- carry a position_id. Non-driving positions — dispatch, planning, the front
-- office — are held by people who log in, so the title sits on the user's
-- membership in the organisation rather than on the user: a position belongs
-- to one organisation, and the same person may hold different titles in two.
ALTER TABLE "user_organization_memberships"
    ADD COLUMN IF NOT EXISTS "position_id" varchar(100);

--bun:split
ALTER TABLE "user_organization_memberships"
    ADD CONSTRAINT "fk_user_organization_memberships_position" FOREIGN KEY ("position_id", "organization_id", "business_unit_id") REFERENCES "job_positions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_organization_memberships_position" ON "user_organization_memberships"("organization_id", "business_unit_id", "position_id")
WHERE
    "position_id" IS NOT NULL;
