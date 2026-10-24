-- A person holds a role in an organization once. Until now only a read-then-insert
-- in the role service kept that true, so two concurrent assignments could both land.
-- Duplicates grant nothing a single row does not, so keep the one that lasts longest
-- (no expiry first, then the latest expiry, then the most recent assignment) and
-- drop the rest before the index makes the rule the database's.
DELETE FROM "user_role_assignments" AS ura
USING (
    SELECT
        "id",
        row_number() OVER (
            PARTITION BY "user_id", "role_id", "organization_id"
            ORDER BY "expires_at" DESC NULLS FIRST, "assigned_at" DESC, "id" DESC
        ) AS rank
    FROM "user_role_assignments"
) AS ranked
WHERE ura."id" = ranked."id"
  AND ranked.rank > 1;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_user_role_assignments_user_role_org"
    ON "user_role_assignments"("user_id", "role_id", "organization_id");
