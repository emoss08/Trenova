--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The scoped Idempotency-Key a create request carried (a SHA-256 of the tenant, the
-- caller and the client's key). The middleware replays a retried request from Redis;
-- this index is what still holds when Redis is unavailable or the replay record has
-- expired, so one key books one shipment.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "idempotency_key" varchar(64);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipments_idempotency_key" ON "shipments"("organization_id", "business_unit_id", "idempotency_key")
WHERE
    "idempotency_key" IS NOT NULL;
