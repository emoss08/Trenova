-- An archive load signs an imported audit trail again under this install's key. When the trail
-- could not be verified where it was signed (its key is not held here, it was unsigned, or it
-- does not verify), each seal and AI audit row it signs carries hash version 3 (signed here) or
-- 4 (plain SHA-256 on an install without a key): imported, not verified at its source. The
-- version is part of what each hash covers, so the mark cannot be removed without breaking the
-- chain, and the verifications report such a chain as ImportedUnverified rather than Verified.
ALTER TABLE "audit_seals"
    DROP CONSTRAINT "ck_audit_seals_hash_version",
    ADD CONSTRAINT "ck_audit_seals_hash_version" CHECK ("hash_version" IN (1, 2, 3, 4));

ALTER TABLE "audit_chain_heads"
    DROP CONSTRAINT "ck_audit_chain_heads_verification",
    ADD CONSTRAINT "ck_audit_chain_heads_verification" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Broken', 'KeyMissing', 'ImportedUnverified'));

ALTER TABLE "ai_audit_events"
    DROP CONSTRAINT "ck_ai_audit_events_hash_version",
    DROP CONSTRAINT "ck_ai_audit_events_signed_key",
    ADD CONSTRAINT "ck_ai_audit_events_hash_version" CHECK ("hash_version" IN (1, 2, 3, 4)),
    ADD CONSTRAINT "ck_ai_audit_events_signed_key" CHECK (("hash_version" IN (1, 3)) = ("hash_key_id" IS NOT NULL));

ALTER TABLE "ai_audit_chain_heads"
    DROP CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status",
    ADD CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Mismatch', 'KeyMissing', 'ImportedUnverified'));
