-- Drop the event trigger and its function first
DROP EVENT TRIGGER IF EXISTS auto_replica_identity_full;

DROP FUNCTION IF EXISTS set_replica_identity_full();

-- Restore REPLICA IDENTITY DEFAULT for all user tables
DO $$
DECLARE
    table_name text;
BEGIN
    FOR table_name IN
    SELECT
        tablename
    FROM
        pg_tables
    WHERE
        schemaname = 'public'
        AND tablename NOT LIKE 'bun_%'
        AND tablename NOT LIKE 'pg_%' LOOP
            EXECUTE format('ALTER TABLE %I REPLICA IDENTITY DEFAULT', table_name);
            RAISE NOTICE 'Restored REPLICA IDENTITY DEFAULT for table: %', table_name;
        END LOOP;
END
$$;

