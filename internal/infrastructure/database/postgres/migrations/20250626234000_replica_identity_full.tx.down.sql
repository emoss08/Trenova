-- Rollback REPLICA IDENTITY FULL to DEFAULT for all user tables
-- This restores the default behavior where only primary key columns
-- are included in the 'before' state for UPDATE and DELETE events

DO $$
DECLARE
    table_name text;
BEGIN
    -- Loop through all user tables in the public schema
    -- Exclude system tables and migration tables
    FOR table_name IN 
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename NOT LIKE 'bun_%'  -- Exclude migration tables
        AND tablename NOT LIKE 'pg_%'   -- Exclude any PostgreSQL system tables
    LOOP
        -- Restore REPLICA IDENTITY DEFAULT for each table
        EXECUTE format('ALTER TABLE %I REPLICA IDENTITY DEFAULT', table_name);
        RAISE NOTICE 'Restored REPLICA IDENTITY DEFAULT for table: %', table_name;
    END LOOP;
END
$$;