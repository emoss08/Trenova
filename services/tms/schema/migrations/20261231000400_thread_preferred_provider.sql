-- A person choosing which model answers them.
--
-- The router already takes a preferred provider and already scopes its
-- candidates to the tenant, to enabled rows, and to the ones assigned the task,
-- so a choice can never reach another organization's endpoint or an untrusted
-- one on work that touches the ledger. What was missing was a place for the
-- choice to live: it came only from agent_definitions.preferred_provider_id,
-- which an administrator sets for everyone.
--
-- It lives on the thread rather than the message so it survives a reload and
-- follows the conversation. What actually served each turn is already recorded
-- per message, which is what lets the reader see when a preference fell through.
--
-- No foreign key: a provider that is deleted should leave the thread working on
-- the usual priority order rather than failing, and the router already ignores
-- a preference it cannot resolve.

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "preferred_provider_id" VARCHAR(100);
