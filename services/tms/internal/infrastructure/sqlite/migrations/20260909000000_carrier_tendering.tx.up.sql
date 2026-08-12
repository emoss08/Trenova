-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260909000000_carrier_tendering.tx.up.sql

CREATE TABLE IF NOT EXISTS "routing_guides"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "origin_location_id" TEXT,
    "destination_location_id" TEXT,
    "origin_city" TEXT,
    "origin_state" TEXT,
    "destination_city" TEXT,
    "destination_state" TEXT,
    "specificity" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_routing_guides" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_routing_guides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_routing_guides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_routing_guides_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_routing_guides_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_routing_guides_specificity" CHECK ("specificity" IN (1, 2, 3)),
    CONSTRAINT "chk_routing_guides_origin_predicate" CHECK (("origin_location_id" IS NOT NULL AND "origin_city" IS NULL AND "origin_state" IS NULL) OR ("origin_location_id" IS NULL AND "origin_state" IS NOT NULL)),
    CONSTRAINT "chk_routing_guides_destination_predicate" CHECK (("destination_location_id" IS NOT NULL AND "destination_city" IS NULL AND "destination_state" IS NULL) OR ("destination_location_id" IS NULL AND "destination_state" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_routing_guides_match" ON "routing_guides" ("organization_id", "business_unit_id", "status", "specificity" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_routing_guides_origin_location" ON "routing_guides" ("origin_location_id", "organization_id")WHERE
    "origin_location_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "routing_guide_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "routing_guide_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "rank" INTEGER NOT NULL,
    "rate_method" TEXT NOT NULL DEFAULT 'Flat',
    "rate" REAL NOT NULL DEFAULT 0,
    "offer_ttl_seconds" INTEGER NOT NULL,
    "channel" TEXT NOT NULL DEFAULT 'Email',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_routing_guide_entries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_routing_guide_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_routing_guide_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_routing_guide_entries_guide" FOREIGN KEY ("routing_guide_id", "business_unit_id", "organization_id") REFERENCES "routing_guides"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_routing_guide_entries_carrier" FOREIGN KEY ("carrier_id", "business_unit_id", "organization_id") REFERENCES "carriers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_routing_guide_entries_rank" CHECK ("rank" >= 1),
    CONSTRAINT "chk_routing_guide_entries_rate" CHECK ("rate" >= 0),
    CONSTRAINT "chk_routing_guide_entries_ttl" CHECK ("offer_ttl_seconds" BETWEEN 300 AND 604800)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_routing_guide_entries_rank" ON "routing_guide_entries" ("routing_guide_id", "rank", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_routing_guide_entries_carrier" ON "routing_guide_entries" ("carrier_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_edi_channels"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "partner_document_profile_id" TEXT,
    "communication_profile_id" TEXT,
    "scac_override" TEXT,
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_edi_channels" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_edi_channels_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_carrier" FOREIGN KEY ("carrier_id", "business_unit_id", "organization_id") REFERENCES "carriers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_edi_channels_document_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_carrier_edi_channels_communication_profile" FOREIGN KEY ("communication_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_communication_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_edi_channels_default" ON "carrier_edi_channels" ("carrier_id", "organization_id", "business_unit_id")WHERE
    "is_default";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_edi_channels_partner" ON "carrier_edi_channels" ("carrier_id", "edi_partner_id", "organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "tenders"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "routing_guide_id" TEXT,
    "mode" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "current_rank" INTEGER NOT NULL DEFAULT 0,
    "workflow_id" TEXT,
    "created_by_id" TEXT,
    "canceled_by_id" TEXT,
    "cancellation_reason" TEXT,
    "accepted_offer_id" TEXT,
    "accepted_at" INTEGER,
    "exhausted_at" INTEGER,
    "canceled_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_tenders" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tenders_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_routing_guide" FOREIGN KEY ("routing_guide_id", "business_unit_id", "organization_id") REFERENCES "routing_guides"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_tenders_live_move" ON "tenders" ("shipment_move_id", "organization_id", "business_unit_id")WHERE
    "status" IN ('Active', 'NeedsReview');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tenders_shipment" ON "tenders" ("shipment_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tenders_status" ON "tenders" ("status", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tenders_workflow" ON "tenders" ("workflow_id")WHERE
    "workflow_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "tender_offers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "tender_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "rank" INTEGER NOT NULL,
    "rate_method" TEXT NOT NULL DEFAULT 'Flat',
    "rate" REAL NOT NULL DEFAULT 0,
    "offer_ttl_seconds" INTEGER NOT NULL,
    "channel" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "recipient_email" TEXT,
    "sent_at" INTEGER,
    "expires_at" INTEGER,
    "responded_at" INTEGER,
    "response_source" TEXT,
    "decline_reason" TEXT,
    "delivery_error" TEXT,
    "edi_partner_id" TEXT,
    "edi_message_id" TEXT,
    "late_response_action" TEXT,
    "late_response_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_tender_offers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tender_offers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offers_tender" FOREIGN KEY ("tender_id", "business_unit_id", "organization_id") REFERENCES "tenders"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offers_carrier" FOREIGN KEY ("carrier_id", "business_unit_id", "organization_id") REFERENCES "carriers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_tender_offers_edi_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_tender_offers_rank" CHECK ("rank" >= 1),
    CONSTRAINT "chk_tender_offers_rate" CHECK ("rate" >= 0),
    CONSTRAINT "chk_tender_offers_ttl" CHECK ("offer_ttl_seconds" BETWEEN 300 AND 604800)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tender_offers_tender_rank" ON "tender_offers" ("tender_id", "rank", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tender_offers_carrier" ON "tender_offers" ("carrier_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tender_offers_sent_expiry" ON "tender_offers" ("expires_at")WHERE
    "status" = 'Sent';

--bun:split

CREATE TABLE IF NOT EXISTS "tender_offer_tokens"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "tender_offer_id" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "used_at" INTEGER,
    "revoked_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_tender_offer_tokens" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tender_offer_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offer_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offer_tokens_offer" FOREIGN KEY ("tender_offer_id", "business_unit_id", "organization_id") REFERENCES "tender_offers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_tender_offer_tokens_hash" ON "tender_offer_tokens" ("token_hash");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tender_offer_tokens_offer" ON "tender_offer_tokens" ("tender_offer_id", "organization_id");
