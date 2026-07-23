CREATE TABLE IF NOT EXISTS dash_controls(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    require_load_acknowledgment BOOLEAN NOT NULL DEFAULT TRUE,
    allow_load_refusals BOOLEAN NOT NULL DEFAULT TRUE,
    allow_stop_actions BOOLEAN NOT NULL DEFAULT TRUE,
    allow_load_document_upload BOOLEAN NOT NULL DEFAULT TRUE,
    allow_load_comments BOOLEAN NOT NULL DEFAULT TRUE,
    show_load_pay BOOLEAN NOT NULL DEFAULT TRUE,
    show_pay_estimates BOOLEAN NOT NULL DEFAULT TRUE,
    allow_expense_submission BOOLEAN NOT NULL DEFAULT TRUE,
    require_expense_receipt BOOLEAN NOT NULL DEFAULT FALSE,
    allow_settlement_disputes BOOLEAN NOT NULL DEFAULT TRUE,
    allow_profile_document_upload BOOLEAN NOT NULL DEFAULT TRUE,
    allow_contact_info_edit BOOLEAN NOT NULL DEFAULT TRUE,
    allow_pto_requests BOOLEAN NOT NULL DEFAULT TRUE,
    send_credential_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    enable_detention_alerts BOOLEAN NOT NULL DEFAULT TRUE,
    detention_alert_threshold_minutes INTEGER NOT NULL DEFAULT 120,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_dash_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_dash_controls_detention_threshold CHECK (detention_alert_threshold_minutes BETWEEN 15 AND 1440),
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_dash_controls_tenant ON dash_controls(organization_id, business_unit_id);
