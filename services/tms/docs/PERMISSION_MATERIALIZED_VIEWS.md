# Permission Materialized Views

## Overview

The permission system uses PostgreSQL materialized views to provide fast permission lookups. The primary materialized view is `user_effective_policies`, which pre-computes all effective permissions for each user in each organization.

## Materialized View: `user_effective_policies`

### Purpose

This view combines:

- User organization memberships
- Role assignments
- Role policy assignments
- Direct policy assignments
- Policy scoping rules

It provides instant permission lookups without expensive JOIN operations on every request.

### Structure

```sql
SELECT
    user_id,
    organization_id,
    business_unit_id,
    policy_id,
    policy_name,
    effect,
    priority,
    resources,
    scope,
    assignment_type,
    source_id
FROM user_effective_policies
WHERE user_id = ? AND organization_id = ?;
```

## Auto-Refresh Mechanism

### Triggers

The materialized view automatically refreshes when changes occur to:

1. **`policies` table** - When policies are created, updated, or deleted
2. **`roles` table** - When roles are created, updated, or deleted
3. **`user_organization_memberships` table** - When user memberships change
4. **`user_organization_roles` table** - When role assignments change

### Trigger Function

```sql
CREATE OR REPLACE FUNCTION trigger_refresh_user_policies()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Refresh the materialized view immediately
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_policies;

    -- Also send notification for any listeners
    PERFORM pg_notify('permission_refresh',
        json_build_object(
            'table', TG_TABLE_NAME,
            'operation', TG_OP,
            'timestamp', EXTRACT(epoch FROM CURRENT_TIMESTAMP)
        )::text
    );

    RETURN COALESCE(NEW, OLD);
END;
$$
LANGUAGE plpgsql;
```

### Concurrent Refresh

The view uses `REFRESH MATERIALIZED VIEW CONCURRENTLY`, which:

- ✅ Allows queries to continue during refresh
- ✅ Only updates changed rows
- ✅ Requires unique index (we have: `idx_user_effective_policies_unique`)
- ⚠️ Takes slightly longer than non-concurrent refresh

## Manual Refresh

### When to Manually Refresh

Manual refresh is needed if:

- Database was restored from backup
- Triggers were disabled temporarily
- Data was bulk-loaded without triggers
- View appears out of sync

### How to Refresh

#### Option 1: Using SQL Function

```bash
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c "SELECT refresh_user_effective_policies();"
```

#### Option 2: Direct SQL

```bash
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c "REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_policies;"
```

#### Option 3: From Go Code

```go
_, err := db.Exec(ctx, "SELECT refresh_user_effective_policies()")
```

### Refresh for Specific User

```sql
SELECT refresh_user_permissions('usr_xxx', 'org_xxx');
```

Note: Currently this refreshes the entire view, but could be optimized for partial refresh in the future.

## Monitoring

### Check View Status

```sql
-- Count total permissions
SELECT COUNT(*) as total_permissions,
       COUNT(DISTINCT user_id) as users,
       COUNT(DISTINCT policy_id) as policies
FROM user_effective_policies;

-- Check specific user
SELECT *
FROM user_effective_policies
WHERE user_id = 'usr_xxx'
  AND organization_id = 'org_xxx';

-- Check for admin user
SELECT u.username, uep.*
FROM user_effective_policies uep
JOIN users u ON u.id = uep.user_id
WHERE u.username = 'admin';
```

### Check Last Refresh Time

The view itself doesn't track refresh time, but you can check trigger activity:

```sql
-- Check pg_notify events (requires logging)
SELECT * FROM pg_stat_activity
WHERE query LIKE '%permission_refresh%';
```

## Performance Considerations

### Refresh Performance

- **Small datasets (< 1000 users)**: < 100ms
- **Medium datasets (1000-10k users)**: 100-500ms
- **Large datasets (10k-100k users)**: 500ms-2s
- **Very large datasets (100k+ users)**: 2-10s

### Optimization Tips

1. **Indexes**: Ensure all indexes are created:

   ```sql
   \d user_effective_policies
   ```

2. **Concurrent Refresh**: Always use `CONCURRENTLY` in production

3. **Partitioning** (future): Partition by business_unit_id for very large datasets

4. **Batch Changes**: If making many policy changes, consider:
   - Disabling triggers temporarily
   - Making all changes
   - Manually refreshing once
   - Re-enabling triggers

## Troubleshooting

### Problem: "No permissions for admin user"

**Cause**: Materialized view is empty or out of sync

**Solution**:

```bash
# 1. Check if view has data
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT COUNT(*) FROM user_effective_policies;"

# 2. Refresh the view
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT refresh_user_effective_policies();"

# 3. Verify admin has permissions
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT * FROM user_effective_policies WHERE user_id IN (SELECT id FROM users WHERE username = 'admin');"
```

### Problem: "Permissions not updating after role change"

**Cause**: Triggers may not have fired

**Solution**:

```bash
# 1. Check if triggers exist
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT tgname, tgenabled FROM pg_trigger WHERE tgname LIKE '%refresh%';"

# 2. Manually refresh
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT refresh_user_effective_policies();"
```

### Problem: "Refresh is very slow"

**Cause**: Large dataset or missing indexes

**Solution**:

```bash
# 1. Check view size
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT pg_size_pretty(pg_total_relation_size('user_effective_policies'));"

# 2. Analyze the view
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "ANALYZE user_effective_policies;"

# 3. Check for missing indexes
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db -c \
  "SELECT indexname FROM pg_indexes WHERE tablename = 'user_effective_policies';"
```

### Problem: "Concurrent refresh fails"

**Error**: `REFRESH MATERIALIZED VIEW CONCURRENTLY requires a unique index`

**Solution**:

```sql
-- Verify unique index exists
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'user_effective_policies'
  AND indexdef LIKE '%UNIQUE%';

-- If missing, create it:
CREATE UNIQUE INDEX idx_user_effective_policies_unique
ON user_effective_policies(user_id, organization_id, policy_id);
```

## Development vs Production

### Development

- Auto-refresh on every change is fine
- Small datasets = fast refresh
- Can use non-concurrent refresh if needed

### Production

- Always use concurrent refresh
- Consider refresh throttling for high-frequency changes
- Monitor refresh duration
- Set up alerts if refresh takes > 5 seconds

## Future Enhancements

### Planned

- [ ] Partial refresh for specific users/orgs
- [ ] Refresh throttling/debouncing
- [ ] Background job queue for refresh (Temporal)
- [ ] View partitioning by business unit
- [ ] Incremental materialized view (PostgreSQL 13+)

### Monitoring

- [ ] Add refresh duration metrics
- [ ] Add refresh failure alerts
- [ ] Track view staleness
- [ ] Dashboard for permission view health

## Related Files

- **Migration**: `internal/infrastructure/postgres/migrations/20250927053803_permission_materialized_views.tx.up.sql`
- **Down Migration**: `internal/infrastructure/postgres/migrations/20250927053803_permission_materialized_views.tx.down.sql`
- **Policy Repository**: `internal/infrastructure/postgres/repositories/policyrepository/policy.go`
- **Permission Engine**: `internal/core/services/permissionservice/engine.go`

## References

- [PostgreSQL Materialized Views](https://www.postgresql.org/docs/current/rules-materializedviews.html)
- [Concurrent Refresh](https://www.postgresql.org/docs/current/sql-refreshmaterializedview.html)
- [Permission System V2 Documentation](PERMISSION_SYSTEM_V2.md)
