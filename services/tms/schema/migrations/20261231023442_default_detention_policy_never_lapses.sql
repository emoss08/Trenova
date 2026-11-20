-- The organization's default detention policy governs every stop no other policy covers, so it
-- must apply from today on: no end date, and no start date still to come. Saving the policy or
-- choosing it in shipment control now refuses either; a default that already carries one is
-- opened up here so no stop is left without a policy.
UPDATE "detention_policies" AS dp
SET "effective_end_date" = NULL,
    "effective_start_date" = CASE
        WHEN dp."effective_start_date" > EXTRACT(EPOCH FROM now())::bigint THEN NULL
        ELSE dp."effective_start_date"
    END
FROM "shipment_controls" AS sc
WHERE sc."default_detention_policy_id" = dp."id"
    AND sc."organization_id" = dp."organization_id"
    AND sc."business_unit_id" = dp."business_unit_id"
    AND (dp."effective_end_date" IS NOT NULL
        OR dp."effective_start_date" > EXTRACT(EPOCH FROM now())::bigint);
