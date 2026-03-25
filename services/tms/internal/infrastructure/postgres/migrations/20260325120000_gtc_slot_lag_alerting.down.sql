SELECT cron.unschedule('check-slot-lag');

--bun:split
SELECT cron.unschedule('check-inactive-slots');

--bun:split
DROP TABLE IF EXISTS gtc_slot_alerts;

--bun:split
DROP EXTENSION IF EXISTS pg_cron;
