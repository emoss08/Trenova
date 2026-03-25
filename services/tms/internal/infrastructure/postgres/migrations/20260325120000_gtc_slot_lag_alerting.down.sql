DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule('check-slot-lag');
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.unschedule('check-inactive-slots');
    END IF;
END $$;

--bun:split
DROP TABLE IF EXISTS gtc_slot_alerts;

--bun:split
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        DROP EXTENSION IF EXISTS pg_cron;
    END IF;
END $$;
