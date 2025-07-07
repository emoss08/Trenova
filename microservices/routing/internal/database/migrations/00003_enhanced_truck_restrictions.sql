-- Add enhanced truck restriction fields to edges table
ALTER TABLE edges ADD COLUMN IF NOT EXISTS max_width DOUBLE PRECISION DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS max_length DOUBLE PRECISION DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS max_axle_load DOUBLE PRECISION DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS hazmat_allowed BOOLEAN DEFAULT true;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS toll_road BOOLEAN DEFAULT false;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS truck_speed_limit INTEGER DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS bridge_max_weight DOUBLE PRECISION DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS tunnel_max_height DOUBLE PRECISION DEFAULT 0;
ALTER TABLE edges ADD COLUMN IF NOT EXISTS time_restrictions JSONB;

-- Create indexes for common truck restriction queries
CREATE INDEX IF NOT EXISTS idx_edges_truck_restrictions ON edges(truck_allowed, max_height, max_weight) WHERE truck_allowed = true;
CREATE INDEX IF NOT EXISTS idx_edges_hazmat ON edges(hazmat_allowed) WHERE hazmat_allowed = true;
CREATE INDEX IF NOT EXISTS idx_edges_toll ON edges(toll_road) WHERE toll_road = true;

-- Add table for tracking regional data imports
CREATE TABLE IF NOT EXISTS import_regions (
    id SERIAL PRIMARY KEY,
    region_name VARCHAR(50) NOT NULL,
    min_lat DOUBLE PRECISION NOT NULL,
    max_lat DOUBLE PRECISION NOT NULL,
    min_lon DOUBLE PRECISION NOT NULL,
    max_lon DOUBLE PRECISION NOT NULL,
    imported_at TIMESTAMP NOT NULL DEFAULT NOW(),
    node_count BIGINT,
    edge_count BIGINT,
    data_version VARCHAR(50),
    UNIQUE(region_name)
);

-- Add table for preferred truck routes
CREATE TABLE IF NOT EXISTS truck_route_preferences (
    id SERIAL PRIMARY KEY,
    edge_id BIGINT REFERENCES edges(id) ON DELETE CASCADE,
    preference_score DOUBLE PRECISION DEFAULT 1.0, -- Higher score = more preferred
    reason VARCHAR(100), -- e.g., 'designated_truck_route', 'avoid_residential'
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_truck_route_preferences_edge ON truck_route_preferences(edge_id);

-- Add statistics tracking
CREATE TABLE IF NOT EXISTS routing_statistics (
    id SERIAL PRIMARY KEY,
    origin_zip VARCHAR(10),
    dest_zip VARCHAR(10),
    distance_miles DOUBLE PRECISION,
    travel_time_minutes DOUBLE PRECISION,
    algorithm_used VARCHAR(50),
    optimization_type VARCHAR(20),
    compute_time_ms INTEGER,
    cache_hit BOOLEAN,
    truck_restrictions_applied JSONB,
    calculated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routing_statistics_zips ON routing_statistics(origin_zip, dest_zip);
CREATE INDEX IF NOT EXISTS idx_routing_statistics_date ON routing_statistics(calculated_at);

-- Function to get import statistics
CREATE OR REPLACE FUNCTION get_import_statistics()
RETURNS TABLE (
    total_nodes BIGINT,
    total_edges BIGINT,
    edges_with_height_restrictions BIGINT,
    edges_with_weight_restrictions BIGINT,
    edges_with_truck_restrictions BIGINT,
    toll_roads BIGINT,
    hazmat_restricted_roads BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*) FROM nodes)::BIGINT as total_nodes,
        (SELECT COUNT(*) FROM edges)::BIGINT as total_edges,
        (SELECT COUNT(*) FROM edges WHERE max_height > 0)::BIGINT as edges_with_height_restrictions,
        (SELECT COUNT(*) FROM edges WHERE max_weight > 0)::BIGINT as edges_with_weight_restrictions,
        (SELECT COUNT(*) FROM edges WHERE truck_allowed = false)::BIGINT as edges_with_truck_restrictions,
        (SELECT COUNT(*) FROM edges WHERE toll_road = true)::BIGINT as toll_roads,
        (SELECT COUNT(*) FROM edges WHERE hazmat_allowed = false)::BIGINT as hazmat_restricted_roads;
END;
$$ LANGUAGE plpgsql;