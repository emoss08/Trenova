--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
--
-- Self-service from the driver portal: policies people sign, and changes they
-- ask to make to their own record.
--
-- An acknowledgement copies what was signed — the version label and the
-- document checksum — because the policy row will move on and the signature
-- must not. A change request stores before-and-after pairs rather than a copy
-- of the wanted record, so a manager sees exactly what is asked and an
-- approval applies exactly that.
CREATE TYPE "worker_policy_audience_enum" AS ENUM(
    'All',
    'Employees',
    'Contractors'
);

--bun:split
CREATE TYPE "profile_change_status_enum" AS ENUM(
    'Pending',
    'Approved',
    'Rejected',
    'Withdrawn'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_policies"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(30) NOT NULL,
    "title" varchar(150) NOT NULL,
    "summary" text,
    -- Either the text itself or an attached document; never neither.
    "body" text,
    "document_id" varchar(100),
    -- The version label is what an acknowledgement records. Changing the
    -- words under a signed policy without changing the label is refused by
    -- the service, because it would make every signature a signature on words
    -- nobody saw.
    "version_label" varchar(30) NOT NULL DEFAULT '1',
    "requires_signature" boolean NOT NULL DEFAULT TRUE,
    "applies_to" worker_policy_audience_enum NOT NULL DEFAULT 'All',
    "effective_from" bigint NOT NULL,
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policies_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_policies_content" CHECK ("document_id" IS NOT NULL OR ("body" IS NOT NULL AND length(btrim("body")) > 0))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_policies_code" ON "worker_policies"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_policies_active" ON "worker_policies"("organization_id", "business_unit_id", "status", "effective_from");

--bun:split
CREATE TABLE IF NOT EXISTS "worker_policy_acknowledgements"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "policy_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "version_label" varchar(30) NOT NULL,
    "acknowledged_at" bigint NOT NULL,
    -- The typed name, the address and client it came from, and the moment: a
    -- record that a specific person, from a specific place, at a specific
    -- time, agreed to a specific text.
    "signature_name" varchar(150),
    "signature_ip" varchar(64),
    "signature_user_agent" varchar(255),
    "document_checksum" varchar(64),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_policy_acknowledgements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_policy_acknowledgements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_policy" FOREIGN KEY ("policy_id", "organization_id", "business_unit_id") REFERENCES "worker_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_policy_acknowledgements_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- One signature per person per version. Signing the same words twice adds
-- nothing, and a second row would be the one an audit asks about.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_policy_acknowledgements_version" ON "worker_policy_acknowledgements"("organization_id", "business_unit_id", "policy_id", "worker_id", "version_label");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_policy_acknowledgements_worker" ON "worker_policy_acknowledgements"("organization_id", "business_unit_id", "worker_id", "acknowledged_at" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_profile_change_requests"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "status" profile_change_status_enum NOT NULL DEFAULT 'Pending',
    -- Before-and-after pairs for the fields being asked about, nothing else.
    "changes" jsonb NOT NULL,
    "note" varchar(500),
    "submitted_at" bigint NOT NULL,
    "decided_at" bigint,
    "decided_by_id" varchar(100),
    "decision_note" varchar(500),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_profile_change_requests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_profile_change_requests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profile_change_requests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profile_change_requests_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- One request waiting per person. A second one would leave a manager deciding
-- two versions of the same address.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_profile_change_requests_open" ON "worker_profile_change_requests"("organization_id", "business_unit_id", "worker_id")
WHERE
    "status" = 'Pending';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_profile_change_requests_queue" ON "worker_profile_change_requests"("organization_id", "business_unit_id", "status", "submitted_at" DESC);

--bun:split
-- Whether a driver's own contact edits wait on somebody. Off keeps today's
-- behaviour, where the edit lands straight on the record.
ALTER TABLE "dash_controls"
    ADD COLUMN IF NOT EXISTS "require_contact_change_approval" boolean NOT NULL DEFAULT FALSE;
