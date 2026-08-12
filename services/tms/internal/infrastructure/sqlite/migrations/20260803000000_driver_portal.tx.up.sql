-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260803000000_driver_portal.tx.up.sql

ALTER TABLE "workers" ADD COLUMN "user_id" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_workers_org_user ON workers (organization_id, user_id)WHERE
    user_id IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS worker_portal_invitations(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "expires_at" INTEGER NOT NULL,
    "invited_by_id" TEXT NOT NULL,
    "accepted_at" INTEGER,
    "accepted_user_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_worker_portal_invitations PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_worker_portal_invitations_status CHECK (status IN ('Pending', 'Accepted', 'Revoked')),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (accepted_user_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_portal_invitations_token_hash ON worker_portal_invitations (token_hash);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_portal_invitations_pending ON worker_portal_invitations (organization_id, business_unit_id, worker_id)WHERE
    status = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_portal_invitations_worker ON worker_portal_invitations (worker_id, organization_id, business_unit_id);

--bun:split

CREATE TABLE IF NOT EXISTS settlement_disputes(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "settlement_id" TEXT NOT NULL,
    "settlement_line_id" TEXT,
    "worker_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "category" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "submitted_by_user_id" TEXT NOT NULL,
    "resolution_note" TEXT,
    "resolution_line_id" TEXT,
    "resolved_by_id" TEXT,
    "resolved_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_settlement_disputes PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_settlement_disputes_status CHECK (status IN ('Open', 'InReview', 'Resolved', 'Denied', 'Withdrawn')),
    CONSTRAINT chk_settlement_disputes_category CHECK (category IN ('MissingPay', 'IncorrectRate', 'IncorrectDeduction', 'MissingReimbursement', 'Other')),
    FOREIGN KEY (settlement_id, organization_id, business_unit_id) REFERENCES driver_settlements(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (settlement_line_id, organization_id, business_unit_id) REFERENCES driver_settlement_lines(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (resolution_line_id, organization_id, business_unit_id) REFERENCES driver_settlement_lines(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (submitted_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (resolved_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_settlement_disputes_settlement ON settlement_disputes (settlement_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_settlement_disputes_worker ON settlement_disputes (worker_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_settlement_disputes_status ON settlement_disputes (organization_id, business_unit_id, status)WHERE
    status IN ('Open', 'InReview');
