-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260507120000_add_shipment_events.tx.up.sql

CREATE TABLE IF NOT EXISTS shipment_events(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "move_id" TEXT,
    "stop_id" TEXT,
    "assignment_id" TEXT,
    "comment_id" TEXT,
    "hold_id" TEXT,
    "type" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'muted',
    "actor_type" TEXT NOT NULL,
    "actor_id" TEXT,
    "actor_label" TEXT,
    "summary" TEXT NOT NULL,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "occurred_at" INTEGER NOT NULL,
    "correlation_id" TEXT,
    CONSTRAINT pk_shipment_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_shipment_events_shipment FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipment_events_tenant_occurred
    ON shipment_events (organization_id, business_unit_id, occurred_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipment_events_shipment_occurred
    ON shipment_events (organization_id, business_unit_id, shipment_id, occurred_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipment_events_type_occurred
    ON shipment_events (organization_id, business_unit_id, type, occurred_at DESC);
