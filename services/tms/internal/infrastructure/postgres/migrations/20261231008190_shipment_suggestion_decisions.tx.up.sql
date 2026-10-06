-- A person's answer to an item in the shipment board's suggestion queue:
-- done, or later. The queue hides what was done and moves what was put off to
-- the back; nothing here changes the shipment itself.
CREATE TABLE IF NOT EXISTS "shipment_suggestion_decisions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "suggestion_key" varchar(200) NOT NULL,
    "decision" varchar(10) NOT NULL,
    "decided_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_shipment_suggestion_decisions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_suggestion_decisions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_suggestion_decisions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_suggestion_decisions_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_shipment_suggestion_decisions_decision" CHECK ("decision" IN ('Done', 'Later'))
);

--bun:split
-- One answer per person per suggestion; answering again replaces it.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipment_suggestion_decisions_key" ON "shipment_suggestion_decisions"("organization_id", "business_unit_id", "user_id", "suggestion_key");

--bun:split
-- The queue reads a person's recent answers, and counts what they handled this shift.
CREATE INDEX IF NOT EXISTS "idx_shipment_suggestion_decisions_recent" ON "shipment_suggestion_decisions"("organization_id", "business_unit_id", "user_id", "decided_at" DESC);

--bun:split
COMMENT ON TABLE "shipment_suggestion_decisions" IS 'A person marking a shipment board suggestion done or for later; keyed by the suggestion it answers';

--bun:split
SELECT trenova_rls.reconcile();
