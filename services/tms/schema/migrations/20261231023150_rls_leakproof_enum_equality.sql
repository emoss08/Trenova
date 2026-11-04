-- Under row-level security PostgreSQL takes a query's own condition into an index scan only when
-- the condition's function is LEAKPROOF: a condition evaluated before the policy must not reveal
-- the rows the policy hides. pg_catalog does not mark enum equality LEAKPROOF, so a status filter
-- (status = 'Waived') never reached the status column of an index for trenova_tenant, and the
-- query read every row of the tenant instead: the detention waiver read took 15 ms where the index
-- answers in 1 ms. enum_eq and enum_ne compare two enum value oids and raise no error, so their
-- result is all they reveal.
--
-- Only a superuser can mark a function LEAKPROOF, and pg_dump and pg_upgrade do not carry the mark,
-- so trenova-db calls mark_leakproof_operators() at the end of every run (Options.Ensure in
-- schema/migrations). Under a migrator that is not a superuser it changes nothing; the application
-- warns at startup and `trenova db rls status` reports it.

CREATE OR REPLACE FUNCTION trenova_rls.leakproof_functions()
    RETURNS regprocedure[]
    LANGUAGE sql
    STABLE
    PARALLEL SAFE
AS $$
    SELECT ARRAY[
        'pg_catalog.enum_eq(anyenum, anyenum)'::regprocedure,
        'pg_catalog.enum_ne(anyenum, anyenum)'::regprocedure
    ]
$$;

CREATE OR REPLACE FUNCTION trenova_rls.leakproof_operators()
    RETURNS boolean
    LANGUAGE sql
    STABLE
    PARALLEL SAFE
AS $$
    SELECT bool_and(p.proleakproof)
    FROM pg_catalog.pg_proc p
    WHERE p.oid = ANY (trenova_rls.leakproof_functions())
$$;

CREATE OR REPLACE FUNCTION trenova_rls.mark_leakproof_operators()
    RETURNS boolean
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    fn regprocedure;
BEGIN
    IF current_setting('is_superuser') = 'on' THEN
        FOREACH fn IN ARRAY trenova_rls.leakproof_functions() LOOP
            IF NOT (SELECT p.proleakproof FROM pg_proc p WHERE p.oid = fn) THEN
                EXECUTE format('ALTER FUNCTION %s LEAKPROOF', fn);
            END IF;
        END LOOP;
    END IF;

    RETURN trenova_rls.leakproof_operators();
END
$$;

REVOKE ALL ON FUNCTION trenova_rls.mark_leakproof_operators() FROM PUBLIC;

GRANT EXECUTE ON FUNCTION trenova_rls.leakproof_functions(), trenova_rls.leakproof_operators() TO trenova_tenant, trenova_rls_bypass;
