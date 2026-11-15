-- A managed instance can send its system email (invitations, password resets, notifications)
-- through a sender Trenova supplies. The operator delivers it from the control plane; one row,
-- replaced when the credential is rotated and removed when Trenova withdraws it. The API key and
-- the webhook secret are sealed by the TMS's own encryption and never leave the instance.
CREATE TABLE IF NOT EXISTS "system_email_senders"(
    "id" varchar(20) NOT NULL,
    "provider" varchar(20) NOT NULL,
    "from_address" varchar(254) NOT NULL,
    "from_name" varchar(40) NOT NULL,
    "api_key_sealed" text NOT NULL,
    "webhook_secret_sealed" text,
    "fingerprint" varchar(64) NOT NULL,
    "received_at" bigint NOT NULL,
    CONSTRAINT "pk_system_email_senders" PRIMARY KEY ("id"),
    CONSTRAINT "ck_system_email_senders_singleton" CHECK ("id" = 'current'),
    CONSTRAINT "ck_system_email_senders_provider" CHECK ("provider" IN ('Resend', 'Postmark')),
    CONSTRAINT "ck_system_email_senders_fingerprint" CHECK ("fingerprint" ~ '^[0-9a-f]{64}$')
);

COMMENT ON TABLE "system_email_senders" IS 'The sender Trenova supplies a managed instance for its system email, as the control plane last sent it';

COMMENT ON COLUMN "system_email_senders"."fingerprint" IS 'The SHA-256 the operator compares to decide whether to send the credential again';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('system_email_senders', 'The instance-wide system email sender the control plane sets for a managed instance; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- Every message names what it is and who carried it. A message Trenova's sender carried has no
-- email profile, and only a system email may be carried by it.
ALTER TABLE "email_messages"
    ADD COLUMN IF NOT EXISTS "type" varchar(60),
    ADD COLUMN IF NOT EXISTS "kind" varchar(20),
    ADD COLUMN IF NOT EXISTS "sender" varchar(20) NOT NULL DEFAULT 'Organization',
    ALTER COLUMN "profile_id" DROP NOT NULL;

ALTER TABLE "email_messages"
    ADD CONSTRAINT "ck_email_messages_kind" CHECK ("kind" IS NULL OR "kind" IN ('System', 'CustomerFacing')),
    ADD CONSTRAINT "ck_email_messages_sender" CHECK (
        ("sender" = 'Organization' AND "profile_id" IS NOT NULL)
        OR ("sender" = 'Trenova' AND "profile_id" IS NULL AND "kind" = 'System')
    );

COMMENT ON COLUMN "email_messages"."sender" IS 'Organization: the organization''s own email profile carried it; Trenova: the system sender Trenova supplies a managed instance';

CREATE INDEX IF NOT EXISTS "idx_email_messages_trenova_provider_id"
    ON "email_messages"("provider", "provider_message_id")
    WHERE "sender" = 'Trenova';
