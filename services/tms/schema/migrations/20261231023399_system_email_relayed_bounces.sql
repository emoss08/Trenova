-- Bounces and spam complaints for email Trenova's sender carries no longer reach the instance
-- through a webhook of its own: the provider account is Trenova's, so its webhook posts to the
-- control plane, which verifies each delivery and the operator relays it to the instance that
-- sent the message. The instance holds no webhook secret, and the fingerprint no longer covers
-- one, so a stored sender is removed: its fingerprint was computed the old way, and the
-- operator puts the sender again on its next tick.
ALTER TABLE "system_email_senders"
    DROP COLUMN IF EXISTS "webhook_secret_sealed";

DELETE FROM "system_email_senders";

COMMENT ON TABLE "system_email_senders" IS 'The sender Trenova supplies a managed instance for its system email, as the control plane last sent it: provider, from address and name, and the API key sealed by the instance''s own encryption';
