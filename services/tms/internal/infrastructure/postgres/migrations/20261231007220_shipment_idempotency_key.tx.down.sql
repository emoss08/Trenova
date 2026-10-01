--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "uq_shipments_idempotency_key";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "idempotency_key";
