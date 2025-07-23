-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

-- +goose Up
-- +goose StatementBegin
-- Insert test nodes for California zip codes (only for development)
INSERT INTO nodes (id, location) VALUES 
(1, ST_SetSRID(ST_MakePoint(-118.2437, 34.0522), 4326)), -- Los Angeles (90001)
(2, ST_SetSRID(ST_MakePoint(-122.4194, 37.7749), 4326)) -- San Francisco (94102)
ON CONFLICT (id) DO NOTHING;

-- Insert test edges (simplified direct route)
INSERT INTO edges (from_node_id, to_node_id, distance, travel_time, truck_allowed) VALUES
(1, 2, 615000, 21600, true), -- ~615km, ~6 hours
(2, 1, 615000, 21600, true) -- Reverse direction
ON CONFLICT DO NOTHING;

-- Map zip codes to nodes
INSERT INTO zip_nodes (zip_code, node_id, centroid, state, city) VALUES
('90001', 1, ST_SetSRID(ST_MakePoint(-118.2437, 34.0522), 4326), 'CA', 'Los Angeles'),
('94102', 2, ST_SetSRID(ST_MakePoint(-122.4194, 37.7749), 4326), 'CA', 'San Francisco')
ON CONFLICT (zip_code) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM zip_nodes WHERE zip_code IN ('90001', '94102');
DELETE FROM edges WHERE from_node_id IN (1, 2) AND to_node_id IN (1, 2);
DELETE FROM nodes WHERE id IN (1, 2);
-- +goose StatementEnd