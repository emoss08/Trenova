ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'OpenWeatherMap';

--bun:split
ALTER TYPE integration_category ADD VALUE IF NOT EXISTS 'Weather';
