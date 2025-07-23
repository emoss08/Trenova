-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

-- Script to populate zip_nodes table from OpenDataSoft US zip code data
-- First, create a temporary table to load the CSV data

CREATE TEMP TABLE temp_zip_data (
    zip_code VARCHAR(10),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    city VARCHAR(100),
    state VARCHAR(2),
    county VARCHAR(100),
    timezone VARCHAR(50)
);

-- Load data from CSV (adjust column names based on actual CSV structure)
-- You'll need to run this COPY command with the actual CSV file path
-- COPY temp_zip_data (zip_code, latitude, longitude, city, state, county, timezone)
-- FROM '/path/to/us-zip-codes.csv' 
-- WITH (FORMAT csv, HEADER true, DELIMITER ';');

-- After loading the CSV, populate zip_nodes by finding nearest road node for each zip
INSERT INTO zip_nodes (zip_code, node_id, centroid, state, city)
SELECT DISTINCT ON (z.zip_code)
    z.zip_code,
    n.id as node_id,
    ST_SetSRID(ST_MakePoint(z.longitude, z.latitude), 4326)::geography as centroid,
    z.state,
    z.city
FROM temp_zip_data z
CROSS JOIN LATERAL (
    -- Find the nearest road node to each zip code centroid
    SELECT id, location
    FROM nodes
    ORDER BY location <-> ST_SetSRID(ST_MakePoint(z.longitude, z.latitude), 4326)
    LIMIT 1
) n
WHERE z.state = 'CA'  -- Only California zip codes for now
ON CONFLICT (zip_code) DO UPDATE SET
    node_id = EXCLUDED.node_id,
    centroid = EXCLUDED.centroid,
    state = EXCLUDED.state,
    city = EXCLUDED.city;

-- Show results
SELECT COUNT(*) as total_zips FROM zip_nodes WHERE state = 'CA';