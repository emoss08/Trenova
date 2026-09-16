-- Hand-written mirror of the PostgreSQL migration; SQLite has no INCLUDE clause, so the
-- index carries only its key columns.
-- Source: 20261224000000_lane_heatmap_indexes.up.sql

CREATE INDEX IF NOT EXISTS "idx_shipments_org_bu_created_at" ON "shipments" ("organization_id", "business_unit_id", "created_at");
