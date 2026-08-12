-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260806000000_dash_controls.tx.up.sql

CREATE TABLE IF NOT EXISTS dash_controls(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "require_load_acknowledgment" INTEGER NOT NULL DEFAULT 1,
    "allow_load_refusals" INTEGER NOT NULL DEFAULT 1,
    "allow_stop_actions" INTEGER NOT NULL DEFAULT 1,
    "allow_load_document_upload" INTEGER NOT NULL DEFAULT 1,
    "allow_load_comments" INTEGER NOT NULL DEFAULT 1,
    "show_load_pay" INTEGER NOT NULL DEFAULT 1,
    "show_pay_estimates" INTEGER NOT NULL DEFAULT 1,
    "allow_expense_submission" INTEGER NOT NULL DEFAULT 1,
    "require_expense_receipt" INTEGER NOT NULL DEFAULT 0,
    "allow_settlement_disputes" INTEGER NOT NULL DEFAULT 1,
    "allow_profile_document_upload" INTEGER NOT NULL DEFAULT 1,
    "allow_contact_info_edit" INTEGER NOT NULL DEFAULT 1,
    "allow_pto_requests" INTEGER NOT NULL DEFAULT 1,
    "send_credential_reminders" INTEGER NOT NULL DEFAULT 1,
    "enable_detention_alerts" INTEGER NOT NULL DEFAULT 1,
    "detention_alert_threshold_minutes" INTEGER NOT NULL DEFAULT 120,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_dash_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_dash_controls_detention_threshold CHECK (detention_alert_threshold_minutes BETWEEN 15 AND 1440),
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_dash_controls_tenant ON dash_controls (organization_id, business_unit_id);
