--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A caller that may retry a generation (a tender offer's 204, re-driven after a
-- crash) names it with a key, so the retry finds the document already generated
-- instead of allocating new control numbers and sending the partner a second one.
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "idempotency_key" varchar(255);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_edi_messages_idempotency_key
    ON "edi_messages" ("organization_id", "business_unit_id", "idempotency_key")
    WHERE "idempotency_key" IS NOT NULL;
