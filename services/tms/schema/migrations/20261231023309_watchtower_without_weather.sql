-- Weather alerts are no longer Watchtower items. They are kept where weather
-- is read (the map and the dispatch console), and on the Watchtower they were
-- noise: dozens of alerts a day about no record anyone could act on. The
-- items already raised go with the source; their events and reads follow.
DELETE FROM "watchtower_items"
WHERE "source_kind" = 'WeatherAlert';
