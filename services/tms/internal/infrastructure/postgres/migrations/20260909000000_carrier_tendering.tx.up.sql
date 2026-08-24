CREATE TABLE IF NOT EXISTS "routing_guides"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "status" varchar(50) NOT NULL DEFAULT 'Active',
    "origin_location_id" varchar(100),
    "destination_location_id" varchar(100),
    "origin_city" varchar(100),
    "origin_state" varchar(2),
    "destination_city" varchar(100),
    "destination_state" varchar(2),
    "specificity" smallint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
CREATE INDEX "idx_routing_guides_match" ON "routing_guides"("organization_id", "business_unit_id", "status", "specificity" DESC);

--bun:split
CREATE INDEX "idx_routing_guides_origin_location" ON "routing_guides"("origin_location_id", "organization_id")
WHERE
    "origin_location_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE "routing_guides" IS 'Ranked carrier lists keyed to a freight lane, used to source waterfall tenders';

--bun:split
CREATE TABLE IF NOT EXISTS "routing_guide_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "routing_guide_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "rank" smallint NOT NULL,
    "rate_method" varchar(50) NOT NULL DEFAULT 'Flat',
    "rate" numeric(19, 4) NOT NULL DEFAULT 0,
    "offer_ttl_seconds" integer NOT NULL,
    "channel" varchar(50) NOT NULL DEFAULT 'Email',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
CREATE UNIQUE INDEX "uq_routing_guide_entries_rank" ON "routing_guide_entries"("routing_guide_id", "rank", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_routing_guide_entries_carrier" ON "routing_guide_entries"("carrier_id", "organization_id");

--bun:split
COMMENT ON TABLE "routing_guide_entries" IS 'One ranked carrier within a routing guide, carrying the pre-negotiated rate and offer expiry';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_edi_channels"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "partner_document_profile_id" varchar(100),
    "communication_profile_id" varchar(100),
    "scac_override" varchar(10),
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_edi_channels" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_edi_channels_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_carrier" FOREIGN KEY ("carrier_id", "business_unit_id", "organization_id") REFERENCES "carriers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_edi_channels_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_edi_channels_document_profile" FOREIGN KEY ("partner_document_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_partner_document_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_carrier_edi_channels_communication_profile" FOREIGN KEY ("communication_profile_id", "business_unit_id", "organization_id") REFERENCES "edi_communication_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX "uq_carrier_edi_channels_default" ON "carrier_edi_channels"("carrier_id", "organization_id", "business_unit_id")
WHERE
    "is_default";

--bun:split
CREATE UNIQUE INDEX "uq_carrier_edi_channels_partner" ON "carrier_edi_channels"("carrier_id", "edi_partner_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE "carrier_edi_channels" IS 'Links a carrier to the EDI partner and profiles used to exchange tender documents with it';

--bun:split
CREATE TABLE IF NOT EXISTS "tenders"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "routing_guide_id" varchar(100),
    "mode" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'Active',
    "current_rank" smallint NOT NULL DEFAULT 0,
    "workflow_id" varchar(255),
    "created_by_id" varchar(100),
    "canceled_by_id" varchar(100),
    "cancellation_reason" text,
    "accepted_offer_id" varchar(100),
    "accepted_at" bigint,
    "exhausted_at" bigint,
    "canceled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_tenders" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tenders_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tenders_routing_guide" FOREIGN KEY ("routing_guide_id", "business_unit_id", "organization_id") REFERENCES "routing_guides"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX "uq_tenders_live_move" ON "tenders"("shipment_move_id", "organization_id", "business_unit_id")
WHERE
    "status" IN ('Active', 'NeedsReview');

--bun:split
CREATE INDEX "idx_tenders_shipment" ON "tenders"("shipment_id", "organization_id");

--bun:split
CREATE INDEX "idx_tenders_status" ON "tenders"("status", "organization_id");

--bun:split
CREATE INDEX "idx_tenders_workflow" ON "tenders"("workflow_id")
WHERE
    "workflow_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE "tenders" IS 'One tendering attempt for a shipment move, driven by a routing-guide waterfall or an ad-hoc spot offer set';

--bun:split
CREATE TABLE IF NOT EXISTS "tender_offers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "tender_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "rank" smallint NOT NULL,
    "rate_method" varchar(50) NOT NULL DEFAULT 'Flat',
    "rate" numeric(19, 4) NOT NULL DEFAULT 0,
    "offer_ttl_seconds" integer NOT NULL,
    "channel" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'Pending',
    "recipient_email" varchar(255),
    "sent_at" bigint,
    "expires_at" bigint,
    "responded_at" bigint,
    "response_source" varchar(50),
    "decline_reason" text,
    "delivery_error" text,
    "edi_partner_id" varchar(100),
    "edi_message_id" varchar(100),
    "late_response_action" varchar(50),
    "late_response_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
CREATE INDEX "idx_tender_offers_tender_rank" ON "tender_offers"("tender_id", "rank", "organization_id");

--bun:split
CREATE INDEX "idx_tender_offers_carrier" ON "tender_offers"("carrier_id", "organization_id");

--bun:split
CREATE INDEX "idx_tender_offers_sent_expiry" ON "tender_offers"("expires_at")
WHERE
    "status" = 'Sent';

--bun:split
COMMENT ON TABLE "tender_offers" IS 'One carrier''s offer within a tender, tracking its dispatch channel, expiry, and response';

--bun:split
CREATE TABLE IF NOT EXISTS "tender_offer_tokens"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "tender_offer_id" varchar(100) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "email" varchar(255) NOT NULL,
    "expires_at" bigint NOT NULL,
    "used_at" bigint,
    "revoked_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_tender_offer_tokens" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tender_offer_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offer_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tender_offer_tokens_offer" FOREIGN KEY ("tender_offer_id", "business_unit_id", "organization_id") REFERENCES "tender_offers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "uq_tender_offer_tokens_hash" ON "tender_offer_tokens"("token_hash");

--bun:split
CREATE INDEX "idx_tender_offer_tokens_offer" ON "tender_offer_tokens"("tender_offer_id", "organization_id");

--bun:split
COMMENT ON TABLE "tender_offer_tokens" IS 'Single-use hashed tokens backing the public accept and decline links for an emailed tender offer';
