-- +goose Up
-- +goose StatementBegin
-- Add enhanced truck restriction fields to edges table
ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS max_width double precision DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS max_length double precision DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS max_axle_load double precision DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS hazmat_allowed boolean DEFAULT TRUE;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS toll_road boolean DEFAULT FALSE;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS truck_speed_limit integer DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS bridge_max_weight double precision DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS tunnel_max_height double precision DEFAULT 0;

ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS time_restrictions jsonb;

-- Create indexes for common truck restriction queries
CREATE INDEX IF NOT EXISTS idx_edges_truck_restrictions ON edges(truck_allowed, max_height, max_weight)
WHERE
    truck_allowed = TRUE;

CREATE INDEX IF NOT EXISTS idx_edges_hazmat ON edges(hazmat_allowed)
WHERE
    hazmat_allowed = TRUE;

CREATE INDEX IF NOT EXISTS idx_edges_toll ON edges(toll_road)
WHERE
    toll_road = TRUE;

-- Add table for tracking regional data imports
CREATE TABLE IF NOT EXISTS import_regions(
    id serial PRIMARY KEY,
    region_name varchar(50) NOT NULL,
    min_lat double precision NOT NULL,
    max_lat double precision NOT NULL,
    min_lon double precision NOT NULL,
    max_lon double precision NOT NULL,
    imported_at timestamp NOT NULL DEFAULT NOW(),
    node_count bigint,
    edge_count bigint,
    data_version varchar(50),
    UNIQUE (region_name)
);

-- Add table for preferred truck routes
CREATE TABLE IF NOT EXISTS truck_route_preferences(
    id serial PRIMARY KEY,
    edge_id bigint REFERENCES edges(id) ON DELETE CASCADE,
    preference_score double precision DEFAULT 1.0, -- Higher score = more preferred
    reason varchar(100), -- e.g., 'designated_truck_route', 'avoid_residential'
    created_at timestamp DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_truck_route_preferences_edge ON truck_route_preferences(edge_id);

-- Add statistics tracking
CREATE TABLE IF NOT EXISTS routing_statistics(
    id serial PRIMARY KEY,
    origin_zip varchar(10),
    dest_zip varchar(10),
    distance_miles double precision,
    travel_time_minutes double precision,
    algorithm_used varchar(50),
    optimization_type varchar(20),
    compute_time_ms integer,
    cache_hit boolean,
    truck_restrictions_applied jsonb,
    calculated_at timestamp DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routing_statistics_zips ON routing_statistics(origin_zip, dest_zip);

CREATE INDEX IF NOT EXISTS idx_routing_statistics_date ON routing_statistics(calculated_at);

-- Function to get import statistics
CREATE OR REPLACE FUNCTION get_import_statistics()
    RETURNS TABLE(
        total_nodes bigint,
        total_edges bigint,
        edges_with_height_restrictions bigint,
        edges_with_weight_restrictions bigint,
        edges_with_truck_restrictions bigint,
        toll_roads bigint,
        hazmat_restricted_roads bigint
    )
    AS $$
BEGIN
    RETURN QUERY
    SELECT
(
            SELECT
                COUNT(*)
            FROM
                nodes)::bigint AS total_nodes,
(
        SELECT
            COUNT(*)
        FROM
            edges)::bigint AS total_edges,
(
        SELECT
            COUNT(*)
        FROM
            edges
        WHERE
            max_height > 0)::bigint AS edges_with_height_restrictions,
(
        SELECT
            COUNT(*)
        FROM
            edges
        WHERE
            max_weight > 0)::bigint AS edges_with_weight_restrictions,
(
        SELECT
            COUNT(*)
        FROM
            edges
        WHERE
            truck_allowed = FALSE)::bigint AS edges_with_truck_restrictions,
(
        SELECT
            COUNT(*)
        FROM
            edges
        WHERE
            toll_road = TRUE)::bigint AS toll_roads,
(
        SELECT
            COUNT(*)
        FROM
            edges
        WHERE
            hazmat_allowed = FALSE)::bigint AS hazmat_restricted_roads;
END;
$$
LANGUAGE plpgsql;

-- +goose StatementEnd
