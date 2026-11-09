-- A look back no longer learns from a tool that failed: a refused call is a defect Trenova
-- fixes in the tool, and every organization would otherwise learn it again. The two signal
-- kinds that recorded those failures are gone, so they leave the look backs and the memories
-- that cite them, and a memory a look back kept about a tool is retired (a suggestion
-- dismissed): it described how to call the tool, which the tool now says itself.
UPDATE "agent_reflections"
SET "signals" = COALESCE(
    (
        SELECT jsonb_agg(signal)
        FROM jsonb_array_elements("signals") AS signal
        WHERE signal->>'kind' NOT IN ('ToolFailed', 'ToolRecovered')
    ),
    '[]'::jsonb
)
WHERE "signals" @> '[{"kind": "ToolFailed"}]'::jsonb
   OR "signals" @> '[{"kind": "ToolRecovered"}]'::jsonb;

UPDATE "agent_memories"
SET "evidence" = jsonb_set(
    "evidence",
    '{signals}',
    COALESCE(
        (
            SELECT jsonb_agg(signal)
            FROM jsonb_array_elements_text("evidence"->'signals') AS signal
            WHERE signal NOT IN ('ToolFailed', 'ToolRecovered')
        ),
        '[]'::jsonb
    )
)
WHERE jsonb_typeof("evidence"->'signals') = 'array'
  AND ("evidence"->'signals' ? 'ToolFailed' OR "evidence"->'signals' ? 'ToolRecovered');

UPDATE "agent_memories"
SET "status" = CASE WHEN "status" = 'Suggested' THEN 'Dismissed' ELSE 'Retired' END,
    "retired_at" = EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" = EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint
WHERE "source" = 'Reflection'
  AND "tool_name" IS NOT NULL
  AND "status" IN ('Active', 'Paused', 'Suggested');
