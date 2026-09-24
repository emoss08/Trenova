-- Hand-written: SQLite has no DELETE ... USING, so each delete names the
-- carried threads through a subquery.
-- Source: 20261231006560_carry_import_conversations.tx.down.sql

DROP TABLE IF EXISTS temp."carried_import_threads";

CREATE TEMP TABLE "carried_import_threads" AS
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

DELETE FROM "assistant_artifacts"
WHERE EXISTS (
    SELECT 1
    FROM "carried_import_threads" t
    WHERE t."id" = "assistant_artifacts"."thread_id"
        AND t."organization_id" = "assistant_artifacts"."organization_id"
        AND t."business_unit_id" = "assistant_artifacts"."business_unit_id"
);

DELETE FROM "assistant_messages"
WHERE EXISTS (
    SELECT 1
    FROM "carried_import_threads" t
    WHERE t."id" = "assistant_messages"."thread_id"
        AND t."organization_id" = "assistant_messages"."organization_id"
        AND t."business_unit_id" = "assistant_messages"."business_unit_id"
);

DELETE FROM "assistant_turns"
WHERE EXISTS (
    SELECT 1
    FROM "carried_import_threads" t
    WHERE t."id" = "assistant_turns"."thread_id"
        AND t."organization_id" = "assistant_turns"."organization_id"
        AND t."business_unit_id" = "assistant_turns"."business_unit_id"
);

DELETE FROM "assistant_threads"
WHERE EXISTS (
    SELECT 1
    FROM "carried_import_threads" t
    WHERE t."id" = "assistant_threads"."id"
        AND t."organization_id" = "assistant_threads"."organization_id"
        AND t."business_unit_id" = "assistant_threads"."business_unit_id"
);

DELETE FROM "agent_definitions"
WHERE "system_key" = 'import_assistant'
    AND EXISTS (
        SELECT 1
        FROM "shipment_import_chat_conversations" c
        WHERE "agent_definitions"."id" = 'agdef_' || substr(c."id", 5)
            AND c."organization_id" = "agent_definitions"."organization_id"
            AND c."business_unit_id" = "agent_definitions"."business_unit_id"
    )
    AND NOT EXISTS (
        SELECT 1
        FROM "assistant_threads" th
        WHERE th."agent_definition_id" = "agent_definitions"."id"
            AND th."organization_id" = "agent_definitions"."organization_id"
            AND th."business_unit_id" = "agent_definitions"."business_unit_id"
    );

DROP TABLE IF EXISTS temp."carried_import_threads";
