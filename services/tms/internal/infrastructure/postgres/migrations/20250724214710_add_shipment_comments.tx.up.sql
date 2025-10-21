--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TABLE IF NOT EXISTS "shipment_comments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "comment" text NOT NULL,
    "comment_type" varchar(100),
    "metadata" jsonb DEFAULT '{}' ::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_shipment_comments" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id"),
    CONSTRAINT "fk_shipment_comments_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_shipment_comments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comments_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for shipment comments table
CREATE INDEX "idx_shipment_comments_shipment" ON "shipment_comments"("shipment_id");

CREATE INDEX "idx_shipment_comments_business_unit" ON "shipment_comments"("business_unit_id");

CREATE INDEX "idx_shipment_comments_organization" ON "shipment_comments"("organization_id");

CREATE INDEX "idx_shipment_comments_org_bu_user" ON "shipment_comments"("organization_id", "business_unit_id", "user_id");

CREATE INDEX "idx_shipment_comments_created_updated" ON "shipment_comments"("created_at", "updated_at");

CREATE INDEX "idx_shipment_comments_metadata" ON "shipment_comments" USING gin("metadata");

COMMENT ON TABLE "shipment_comments" IS 'Stores comments for shipments';

--bun:split
-- Create shipment comment mentions table
CREATE TABLE IF NOT EXISTS "shipment_comment_mentions"(
    "id" varchar(100) NOT NULL,
    "comment_id" varchar(100) NOT NULL,
    "mentioned_user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_shipment_comment_mentions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_shipment_comment_mentions_comment" FOREIGN KEY ("comment_id", "business_unit_id", "organization_id", "shipment_id") REFERENCES "shipment_comments"("id", "business_unit_id", "organization_id", "shipment_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_mentions_user" FOREIGN KEY ("mentioned_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_mentions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_shipment_comment_mentions_comment_user" UNIQUE ("comment_id", "mentioned_user_id")
);

--bun:split
-- Indexes for shipment comment mentions table
CREATE INDEX "idx_shipment_comment_mentions_comment" ON "shipment_comment_mentions"("comment_id");

CREATE INDEX "idx_shipment_comment_mentions_user" ON "shipment_comment_mentions"("mentioned_user_id");

CREATE INDEX "idx_shipment_comment_mentions_organization" ON "shipment_comment_mentions"("organization_id");

COMMENT ON TABLE "shipment_comment_mentions" IS 'Tracks users mentioned in shipment comments for notifications';

