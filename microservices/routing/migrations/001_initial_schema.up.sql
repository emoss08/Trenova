--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- Nodes (intersections)
CREATE TABLE IF NOT EXISTS nodes (
    id BIGSERIAL PRIMARY KEY,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    osm_id BIGINT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_nodes_location ON nodes USING GIST(location);
CREATE INDEX idx_nodes_osm_id ON nodes(osm_id);

-- Edges (road segments)
CREATE TABLE IF NOT EXISTS edges (
    id BIGSERIAL PRIMARY KEY,
    from_node_id BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node_id BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    distance DOUBLE PRECISION NOT NULL, -- meters
    travel_time DOUBLE PRECISION NOT NULL, -- seconds
    max_height DOUBLE PRECISION DEFAULT 0, -- meters (0 = no restriction)
    max_weight DOUBLE PRECISION DEFAULT 0, -- kg (0 = no restriction)
    truck_allowed BOOLEAN DEFAULT true,
    road_type VARCHAR(50),
    osm_way_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_edges_from_node ON edges(from_node_id);
CREATE INDEX idx_edges_to_node ON edges(to_node_id);
CREATE INDEX idx_edges_truck_allowed ON edges(truck_allowed);

-- Zip code mappings for quick lookups
CREATE TABLE IF NOT EXISTS zip_nodes (
    zip_code VARCHAR(10) PRIMARY KEY,
    node_id BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    centroid GEOGRAPHY(POINT, 4326) NOT NULL,
    state VARCHAR(2),
    city VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_zip_nodes_state ON zip_nodes(state);

-- Cached routes for frequently requested paths
CREATE TABLE IF NOT EXISTS cached_routes (
    id BIGSERIAL PRIMARY KEY,
    origin_zip VARCHAR(10) NOT NULL,
    dest_zip VARCHAR(10) NOT NULL,
    distance DOUBLE PRECISION NOT NULL, -- miles
    travel_time DOUBLE PRECISION NOT NULL, -- minutes
    route_path JSONB, -- Optional: store the actual path
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP + INTERVAL '48 hours'
);

CREATE INDEX idx_cached_routes_zips ON cached_routes(origin_zip, dest_zip);
CREATE INDEX idx_cached_routes_expires ON cached_routes(expires_at);

-- Function to clean up expired cache entries
CREATE OR REPLACE FUNCTION clean_expired_cache() RETURNS void AS $$
BEGIN
    DELETE FROM cached_routes WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;