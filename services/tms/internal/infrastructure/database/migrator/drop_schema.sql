DO $$
BEGIN
    IF NOT EXISTS(
        SELECT
            1
        FROM
            pg_replication_slots
        WHERE
            slot_name = 'trenova_slot') THEN
    PERFORM
        pg_create_logical_replication_slot('trenova_slot', 'pgoutput');
    RAISE NOTICE 'Created replication slot trenova_slot';
ELSE
    RAISE NOTICE 'Replication slot trenova_slot already exists';
END IF;
END
$$;

DROP SCHEMA public CASCADE;

CREATE SCHEMA public;

GRANT ALL ON SCHEMA public TO public;

DO $$
BEGIN
    IF NOT EXISTS(
        SELECT
            1
        FROM
            pg_roles
        WHERE
            rolname = 'debezium_user') THEN
    CREATE ROLE debezium_user WITH LOGIN REPLICATION PASSWORD 'debezium_admin123@';
    RAISE NOTICE 'Created role debezium_user';
ELSE
    ALTER ROLE debezium_user WITH LOGIN REPLICATION PASSWORD 'debezium_admin123@';
    RAISE NOTICE 'Role debezium_user already exists, updated attributes';
END IF;
END
$$;

GRANT CONNECT ON DATABASE trenova_go_db TO debezium_user;

GRANT USAGE ON SCHEMA public TO debezium_user;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO debezium_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT
SELECT
    ON TABLES TO debezium_user;

DO $$
BEGIN
    IF NOT EXISTS(
        SELECT
            1
        FROM
            pg_publication
        WHERE
            pubname = 'trenova_publication') THEN
    CREATE PUBLICATION trenova_publication FOR ALL TABLES;
    RAISE NOTICE 'Created publication trenova_publication';
ELSE
    RAISE NOTICE 'Publication trenova_publication already exists';
END IF;
END
$$;

