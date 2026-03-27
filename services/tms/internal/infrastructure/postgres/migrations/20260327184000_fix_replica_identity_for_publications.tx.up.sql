DROP EVENT TRIGGER IF EXISTS auto_replica_identity_full;
DROP FUNCTION IF EXISTS set_replica_identity_full();

CREATE OR REPLACE FUNCTION set_replica_identity_for_new_table()
    RETURNS event_trigger
    LANGUAGE plpgsql
AS $$
DECLARE
    obj RECORD;
    obj_schema TEXT;
    obj_table TEXT;
    has_primary_key BOOLEAN;
BEGIN
    FOR obj IN
        SELECT *
        FROM pg_event_trigger_ddl_commands()
    LOOP
        IF obj.command_tag <> 'CREATE TABLE' OR obj.schema_name <> 'public' THEN
            CONTINUE;
        END IF;

        IF obj.object_identity LIKE 'public.bun_%'
           OR obj.object_identity LIKE 'public.pg_%'
           OR obj.object_identity LIKE 'public.gtc_%'
           OR obj.object_identity LIKE 'public.seed_%' THEN
            CONTINUE;
        END IF;

        obj_schema := split_part(obj.object_identity, '.', 1);
        obj_table := split_part(obj.object_identity, '.', 2);

        SELECT EXISTS (
            SELECT 1
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN pg_index i ON i.indrelid = c.oid
            WHERE n.nspname = obj_schema
              AND c.relname = obj_table
              AND i.indisprimary
        )
        INTO has_primary_key;

        IF NOT has_primary_key THEN
            EXECUTE format('ALTER TABLE %s REPLICA IDENTITY FULL', obj.object_identity);
            RAISE NOTICE 'Auto-set REPLICA IDENTITY FULL for: %', obj.object_identity;
        END IF;
    END LOOP;
END;
$$;

CREATE EVENT TRIGGER auto_replica_identity_full ON ddl_command_end
    WHEN TAG IN ('CREATE TABLE')
    EXECUTE FUNCTION set_replica_identity_for_new_table();

DO $$
DECLARE
    tbl RECORD;
BEGIN
    FOR tbl IN
        SELECT c.relname AS table_name
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_index i ON i.indrelid = c.oid
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
          AND c.relreplident = 'f'
          AND i.indisprimary
          AND c.relname NOT LIKE 'bun_%'
          AND c.relname NOT LIKE 'pg_%'
          AND c.relname NOT LIKE 'gtc_%'
          AND c.relname NOT LIKE 'seed_%'
    LOOP
        EXECUTE format('ALTER TABLE %I REPLICA IDENTITY DEFAULT', tbl.table_name);
        RAISE NOTICE 'Set REPLICA IDENTITY DEFAULT on %', tbl.table_name;
    END LOOP;
END;
$$;
