DROP EVENT TRIGGER IF EXISTS auto_replica_identity_full;
DROP FUNCTION IF EXISTS set_replica_identity_for_new_table();

CREATE OR REPLACE FUNCTION set_replica_identity_full()
    RETURNS event_trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    obj record;
BEGIN
    FOR obj IN
    SELECT
        *
    FROM
        pg_event_trigger_ddl_commands()
        LOOP
            IF obj.command_tag = 'CREATE TABLE' AND obj.schema_name = 'public' THEN
                IF obj.object_identity NOT LIKE 'public.bun_%'
                   AND obj.object_identity NOT LIKE 'public.pg_%' THEN
                    EXECUTE format('ALTER TABLE %s REPLICA IDENTITY FULL', obj.object_identity);
                    RAISE NOTICE 'Auto-set REPLICA IDENTITY FULL for: %', obj.object_identity;
                END IF;
            END IF;
        END LOOP;
END;
$$;

CREATE EVENT TRIGGER auto_replica_identity_full ON ddl_command_end
    WHEN TAG IN('CREATE TABLE')
        EXECUTE FUNCTION set_replica_identity_full();

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
