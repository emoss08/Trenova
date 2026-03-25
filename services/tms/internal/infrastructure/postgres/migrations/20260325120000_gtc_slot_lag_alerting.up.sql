CREATE EXTENSION IF NOT EXISTS pg_cron;

--bun:split
CREATE TABLE IF NOT EXISTS gtc_slot_alerts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slot_name text NOT NULL,
    lag_bytes bigint NOT NULL,
    checked_at timestamptz NOT NULL DEFAULT now()
);

--bun:split
SELECT cron.schedule('check-slot-lag', '*/5 * * * *', $$
    INSERT INTO gtc_slot_alerts (slot_name, lag_bytes, checked_at)
    SELECT slot_name,
           pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn),
           now()
    FROM pg_replication_slots
    WHERE pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn) > 5368709120;
$$);

--bun:split
SELECT cron.schedule('check-inactive-slots', '*/5 * * * *', $$
    INSERT INTO gtc_slot_alerts (slot_name, lag_bytes, checked_at)
    SELECT slot_name, 0, now()
    FROM pg_replication_slots
    WHERE NOT active
    AND confirmed_flush_lsn IS NOT NULL;
$$);
