ALTER TABLE "rate_confirmations"
    ADD COLUMN "generated_via" varchar(50) NOT NULL DEFAULT 'Dispatcher',
    ADD COLUMN "source_tender_offer_id" varchar(100),
    ADD COLUMN "confirmed_via" varchar(50),
    ADD COLUMN "confirmed_by_title" varchar(255);

--bun:split
COMMENT ON COLUMN "rate_confirmations"."generated_via" IS 'How the revision came to exist: Dispatcher for the manual flow, TenderAcceptance for the auto-issued agreement';

--bun:split
COMMENT ON COLUMN "rate_confirmations"."confirmed_via" IS 'How the agreement was executed: Dispatcher, PublicSignature, or TenderAcceptance';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_confirmation_tokens"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_confirmation_id" varchar(100) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "email" varchar(255) NOT NULL,
    "expires_at" bigint NOT NULL,
    "used_at" bigint,
    "revoked_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_rate_confirmation_tokens" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_confirmation_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmation_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmation_tokens_rate_confirmation" FOREIGN KEY ("rate_confirmation_id", "business_unit_id", "organization_id") REFERENCES "rate_confirmations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "uq_rate_confirmation_tokens_hash" ON "rate_confirmation_tokens"("token_hash");

--bun:split
CREATE INDEX "idx_rate_confirmation_tokens_rate_confirmation" ON "rate_confirmation_tokens"("rate_confirmation_id", "organization_id");

--bun:split
COMMENT ON TABLE "rate_confirmation_tokens" IS 'Single-use hashed tokens backing the public preview and sign links for an emailed rate confirmation';
