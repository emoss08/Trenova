DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_cron;
EXCEPTION
    WHEN undefined_file OR insufficient_privilege OR feature_not_supported THEN
        RAISE NOTICE 'pg_cron is unavailable, skipping cron schedule setup';
END $$;

--bun:split
CREATE TABLE IF NOT EXISTS gtc_slot_alerts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slot_name text NOT NULL,
    lag_bytes bigint NOT NULL,
    checked_at timestamptz NOT NULL DEFAULT now()
);

--bun:split
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.schedule('check-slot-lag', '*/5 * * * *', $job$
            INSERT INTO gtc_slot_alerts (slot_name, lag_bytes, checked_at)
            SELECT slot_name,
                   pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn),
                   now()
            FROM pg_replication_slots
            WHERE pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn) > 5368709120;
        $job$);
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.schedule('check-inactive-slots', '*/5 * * * *', $job$
            INSERT INTO gtc_slot_alerts (slot_name, lag_bytes, checked_at)
            SELECT slot_name, 0, now()
            FROM pg_replication_slots
            WHERE NOT active
            AND confirmed_flush_lsn IS NOT NULL;
        $job$);
    END IF;
END $$;
