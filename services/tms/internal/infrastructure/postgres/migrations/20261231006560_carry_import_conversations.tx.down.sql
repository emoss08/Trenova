-- Removes what the carry-over wrote and nothing else: the threads whose ids it
-- derived from legacy turns, their messages and artifacts, and the import
-- assistant it created, when no other conversation has come to use it. The
-- legacy conversations were never changed, so they need no restoring.
SET statement_timeout = 0;

CREATE TEMP TABLE "carried_import_threads" ON COMMIT DROP AS
SELECT DISTINCT
    th."id",
    th."organization_id",
    th."business_unit_id"
FROM "assistant_threads" th
JOIN "shipment_import_chat_turns" turns
    ON th."id" = 'athr_' || substr(turns."id", 5)
    AND th."organization_id" = turns."organization_id"
    AND th."business_unit_id" = turns."business_unit_id"
WHERE th."origin" = 'Import';

DELETE FROM "assistant_artifacts" a
USING "carried_import_threads" t
WHERE a."thread_id" = t."id"
    AND a."organization_id" = t."organization_id"
    AND a."business_unit_id" = t."business_unit_id";

DELETE FROM "assistant_messages" m
USING "carried_import_threads" t
WHERE m."thread_id" = t."id"
    AND m."organization_id" = t."organization_id"
    AND m."business_unit_id" = t."business_unit_id";

DELETE FROM "assistant_turns" turns
USING "carried_import_threads" t
WHERE turns."thread_id" = t."id"
    AND turns."organization_id" = t."organization_id"
    AND turns."business_unit_id" = t."business_unit_id";

DELETE FROM "assistant_threads" th
USING "carried_import_threads" t
WHERE th."id" = t."id"
    AND th."organization_id" = t."organization_id"
    AND th."business_unit_id" = t."business_unit_id";

DELETE FROM "agent_definitions" d
WHERE d."system_key" = 'import_assistant'
    AND EXISTS (
        SELECT 1
        FROM "shipment_import_chat_conversations" c
        WHERE d."id" = 'agdef_' || substr(c."id", 5)
            AND c."organization_id" = d."organization_id"
            AND c."business_unit_id" = d."business_unit_id"
    )
    AND NOT EXISTS (
        SELECT 1
        FROM "assistant_threads" th
        WHERE th."agent_definition_id" = d."id"
            AND th."organization_id" = d."organization_id"
            AND th."business_unit_id" = d."business_unit_id"
    );
