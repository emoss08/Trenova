--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE customer_billing_profiles
    ADD COLUMN IF NOT EXISTS "default_biller_id" varchar(100) REFERENCES users(id) ON DELETE SET NULL;
