-- A tenant archive taken for a final archive, a move or the nightly backup states what its source
-- established about its audit trails: each trail sealed through the snapshot, verified at the
-- source, and the number of links it carries. An import that cannot verify a chain itself (the
-- source's key is not held here) and whose own key-free checks of what arrived agree with that
-- statement signs the chain again with hash version 5 (signed here) or 6 (plain SHA-256 on an
-- install without a key): imported, verified at its source as its archive states. The version is
-- part of what each hash covers, like 3 and 4, and the verifications report such a chain as
-- ImportedSourceVerified, never as Verified. The status columns widen to hold that name (raising
-- a varchar's length rewrites nothing).
ALTER TABLE "audit_chain_heads"
    ALTER COLUMN "last_verification_status" TYPE varchar(32);

ALTER TABLE "ai_audit_chain_heads"
    ALTER COLUMN "last_verification_status" TYPE varchar(32);

ALTER TABLE "audit_seals"
    DROP CONSTRAINT "ck_audit_seals_hash_version",
    ADD CONSTRAINT "ck_audit_seals_hash_version" CHECK ("hash_version" IN (1, 2, 3, 4, 5, 6));

ALTER TABLE "audit_chain_heads"
    DROP CONSTRAINT "ck_audit_chain_heads_verification",
    ADD CONSTRAINT "ck_audit_chain_heads_verification" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Broken', 'KeyMissing', 'ImportedUnverified', 'ImportedSourceVerified'));

ALTER TABLE "ai_audit_events"
    DROP CONSTRAINT "ck_ai_audit_events_hash_version",
    DROP CONSTRAINT "ck_ai_audit_events_signed_key",
    ADD CONSTRAINT "ck_ai_audit_events_hash_version" CHECK ("hash_version" IN (1, 2, 3, 4, 5, 6)),
    ADD CONSTRAINT "ck_ai_audit_events_signed_key" CHECK (("hash_version" IN (1, 3, 5)) = ("hash_key_id" IS NOT NULL));

ALTER TABLE "ai_audit_chain_heads"
    DROP CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status",
    ADD CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Mismatch', 'KeyMissing', 'ImportedUnverified', 'ImportedSourceVerified'));
