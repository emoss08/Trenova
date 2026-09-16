-- Hand-written mirror of the PostgreSQL migration.
-- Source: 20261224000000_lane_heatmap_indexes.down.sql

DROP INDEX IF EXISTS "idx_shipments_org_bu_created_at";
