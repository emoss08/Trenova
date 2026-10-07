-- The shipment board's brief for one organization's day: the sentence at the
-- top of the board and the reworded suggested actions, written once before the
-- workday and again only when the organization clears every open issue. The
-- board reads it rather than asking a model on every visit.
CREATE TABLE IF NOT EXISTS "shipment_board_briefs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "brief_date" varchar(10) NOT NULL,
    "generation" integer NOT NULL,
    "trigger" varchar(20) NOT NULL,
    "segments" jsonb NOT NULL DEFAULT '[]',
    "wording" jsonb NOT NULL DEFAULT '{}',
    "facts" jsonb NOT NULL DEFAULT '{}',
    "open_issues" integer NOT NULL DEFAULT 0,
    "narrated" boolean NOT NULL DEFAULT FALSE,
    "model_identifier" varchar(255),
    "generated_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_shipment_board_briefs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_board_briefs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_board_briefs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_shipment_board_briefs_trigger" CHECK ("trigger" IN ('Scheduled', 'OnDemand', 'Cleared')),
    CONSTRAINT "ck_shipment_board_briefs_generation" CHECK ("generation" >= 1)
);

--bun:split
-- One brief per generation of an organization's day; two writers racing for
-- the same generation leave one row.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipment_board_briefs_generation" ON "shipment_board_briefs"("organization_id", "business_unit_id", "brief_date", "generation");

--bun:split
-- The retention sweep removes past days across every tenant.
CREATE INDEX IF NOT EXISTS "idx_shipment_board_briefs_date" ON "shipment_board_briefs"("brief_date");

--bun:split
COMMENT ON TABLE "shipment_board_briefs" IS 'The shipment board brief for an organization''s day, written once before the workday and again when every open issue is cleared';

--bun:split
SELECT trenova_rls.reconcile();
