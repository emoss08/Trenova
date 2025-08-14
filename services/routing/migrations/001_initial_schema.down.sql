--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS cached_routes;
DROP TABLE IF EXISTS zip_nodes;
DROP TABLE IF EXISTS edges;
DROP TABLE IF EXISTS nodes;
DROP FUNCTION IF EXISTS clean_expired_cache();
DROP EXTENSION IF EXISTS postgis_topology;
DROP EXTENSION IF EXISTS postgis;