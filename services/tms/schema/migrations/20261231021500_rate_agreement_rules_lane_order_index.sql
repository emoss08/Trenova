--keel:notx
-- An agreement's lanes are read most specific first, a page or a first few per agreement at a time.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_rate_agreement_rules_lane_order" ON "rate_agreement_rules"("organization_id", "business_unit_id", "rate_agreement_id", "specificity_score" DESC, "id")
WHERE
    "status" = 'Active';
