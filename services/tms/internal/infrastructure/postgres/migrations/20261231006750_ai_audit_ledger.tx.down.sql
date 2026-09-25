DROP TABLE IF EXISTS "ai_audit_projector_state";

--bun:split

DROP TABLE IF EXISTS "ai_audit_exports";

--bun:split

DROP TABLE IF EXISTS "ai_audit_seals";

--bun:split

DROP TABLE IF EXISTS "ai_audit_chain_heads";

--bun:split

DROP TABLE IF EXISTS "ai_audit_events";

--bun:split

DROP FUNCTION IF EXISTS "ai_audit_events_append_only"();
