-- Event trigger to automatically set REPLICA IDENTITY FULL on any new table
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
                -- Skip migration/system tables
                IF obj.object_identity NOT LIKE 'public.bun_%' AND obj.object_identity NOT LIKE 'public.pg_%' THEN
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

