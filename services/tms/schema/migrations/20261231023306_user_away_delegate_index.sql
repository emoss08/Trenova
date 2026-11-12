--keel:notx
-- While someone is away, the person they named to cover works their Desk
-- cases. Every list of cases asks whose cover the reader is; this is that
-- question's index, over the few people away at any time.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_users_away_delegate" ON "users"("business_unit_id", "away_delegate_user_id")
WHERE
    "availability_status" = 'away' AND "away_delegate_user_id" IS NOT NULL;
