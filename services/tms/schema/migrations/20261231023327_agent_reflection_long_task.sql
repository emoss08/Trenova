-- A look back no longer starts because a task took many steps: with nothing a person said or
-- decided in it, there is almost nothing it may keep, and each one cost a model call. The
-- signal kind is gone, so it leaves the look backs and the memories that cite it.
UPDATE "agent_reflections"
SET "signals" = COALESCE(
    (
        SELECT jsonb_agg(signal)
        FROM jsonb_array_elements("signals") AS signal
        WHERE signal->>'kind' <> 'LongTask'
    ),
    '[]'::jsonb
)
WHERE "signals" @> '[{"kind": "LongTask"}]'::jsonb;

UPDATE "agent_memories"
SET "evidence" = jsonb_set(
    "evidence",
    '{signals}',
    COALESCE(
        (
            SELECT jsonb_agg(signal)
            FROM jsonb_array_elements_text("evidence"->'signals') AS signal
            WHERE signal <> 'LongTask'
        ),
        '[]'::jsonb
    )
)
WHERE jsonb_typeof("evidence"->'signals') = 'array'
  AND "evidence"->'signals' ? 'LongTask';
