-- Add any new ENUM types here

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'org_type') THEN
        CREATE TYPE org_type AS ENUM ('ASSET', 'BROKERAGE', 'BOTH');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'lang_type') THEN
        CREATE TYPE lang_type AS ENUM ('en-US', 'es-US');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_type') THEN
        CREATE TYPE status_type AS ENUM ('A', 'I');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_type') THEN
        CREATE TYPE status_type AS ENUM ('A', 'I');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'email_protocol_type') THEN
        CREATE TYPE email_protocol_type AS ENUM ('TLS', 'SSL', 'UNENCRYPTED');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_function_type') THEN
        CREATE TYPE job_function_type AS ENUM (
            'MANAGER',
            'MANAGEMENT_TRAINEE',
            'SUPERVISOR',
            'DISPATCHER',
            'BILLING',
            'FINANCE',
            'SAFETY',
            'DRIVER',
            'MECHANIC',
            'SYS_ADMIN'
        );
    END IF;
END
$$;