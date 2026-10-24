CREATE TABLE IF NOT EXISTS equipment_continuity (
    id VARCHAR(100) PRIMARY KEY,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    equipment_type VARCHAR(20) NOT NULL,
    equipment_id VARCHAR(100) NOT NULL,
    current_location_id VARCHAR(100) NOT NULL,
    previous_continuity_id VARCHAR(100) NULL,
    source_type VARCHAR(32) NOT NULL,
    source_shipment_id VARCHAR(100) NULL,
    source_shipment_move_id VARCHAR(100) NULL,
    source_assignment_id VARCHAR(100) NULL,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    superseded_at BIGINT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint
);

CREATE INDEX IF NOT EXISTS idx_equipment_continuity_equipment
    ON equipment_continuity (organization_id, business_unit_id, equipment_type, equipment_id);

CREATE INDEX IF NOT EXISTS idx_equipment_continuity_shipment
    ON equipment_continuity (organization_id, business_unit_id, source_shipment_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_equipment_continuity_current_unique
    ON equipment_continuity (organization_id, business_unit_id, equipment_type, equipment_id)
    WHERE is_current = TRUE;
