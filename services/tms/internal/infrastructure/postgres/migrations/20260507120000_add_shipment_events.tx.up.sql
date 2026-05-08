CREATE TABLE IF NOT EXISTS shipment_events(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    shipment_id VARCHAR(100) NOT NULL,
    move_id VARCHAR(100),
    stop_id VARCHAR(100),
    assignment_id VARCHAR(100),
    comment_id VARCHAR(100),
    hold_id VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'muted',
    actor_type VARCHAR(20) NOT NULL,
    actor_id VARCHAR(100),
    actor_label VARCHAR(100),
    summary TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at BIGINT NOT NULL,
    correlation_id VARCHAR(100),
    CONSTRAINT pk_shipment_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_shipment_events_shipment FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_events_tenant_occurred
    ON shipment_events(organization_id, business_unit_id, occurred_at DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_events_shipment_occurred
    ON shipment_events(organization_id, business_unit_id, shipment_id, occurred_at DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_events_type_occurred
    ON shipment_events(organization_id, business_unit_id, type, occurred_at DESC);
