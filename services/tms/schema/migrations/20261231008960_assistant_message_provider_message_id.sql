-- An assistant reply keeps the provider's own id for it. When the operator turns on
-- cache diagnostics, the next call to the same provider names it, so the Claude API
-- can say where that call's prompt stopped matching the one before. Nothing else
-- reads it.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "provider_message_id" varchar(200);
