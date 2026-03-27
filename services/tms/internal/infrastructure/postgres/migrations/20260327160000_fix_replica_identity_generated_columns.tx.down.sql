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
          AND c.relreplident = 'f'
    LOOP
        EXECUTE format('ALTER TABLE %I REPLICA IDENTITY DEFAULT', tbl.table_name);
    END LOOP;
END;
$$;
