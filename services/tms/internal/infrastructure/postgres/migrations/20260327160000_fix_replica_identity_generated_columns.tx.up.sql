DO $$
DECLARE
    tbl RECORD;
BEGIN
    FOR tbl IN
        SELECT c.relname AS table_name
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
          AND c.relreplident = 'd'
          AND c.relname NOT LIKE 'bun_%'
          AND c.relname NOT LIKE 'pg_%'
          AND c.relname NOT LIKE 'gtc_%'
          AND c.relname NOT LIKE 'seed_%'
    LOOP
        EXECUTE format('ALTER TABLE %I REPLICA IDENTITY FULL', tbl.table_name);
        RAISE NOTICE 'Set REPLICA IDENTITY FULL on %', tbl.table_name;
    END LOOP;
END;
$$;
