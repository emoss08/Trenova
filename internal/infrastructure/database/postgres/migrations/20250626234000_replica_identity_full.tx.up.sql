-- Set REPLICA IDENTITY FULL for all user tables to enable full row capture in CDC
-- This ensures that UPDATE and DELETE events include all column values in the 'before' state
-- instead of just primary key columns (which is the DEFAULT behavior)

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
        -- Set REPLICA IDENTITY FULL for each table
        EXECUTE format('ALTER TABLE %I REPLICA IDENTITY FULL', table_name);
        RAISE NOTICE 'Set REPLICA IDENTITY FULL for table: %', table_name;
    END LOOP;
END
$$;