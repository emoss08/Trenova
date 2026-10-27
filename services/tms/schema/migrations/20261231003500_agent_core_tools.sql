UPDATE "agent_definitions"
SET "tool_names" = array_remove(
        array_remove(
            array_remove(
                array_remove("tool_names", 'recall_memory'),
                'remember'),
            'raise_exception'),
        'flag_for_manual_review')
WHERE "tool_names" && ARRAY['recall_memory', 'remember', 'raise_exception', 'flag_for_manual_review']::text[];
