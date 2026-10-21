-- The exact acr values an identity provider sends for a multi-factor sign-in.
-- Empty means only an exact multi-factor amr value is trusted.
ALTER TABLE "identity_providers"
    ADD COLUMN IF NOT EXISTS "federated_mfa_acr_values" text[] NOT NULL DEFAULT ARRAY[]::text[];
