-- The formula assistant's tools work on Tariff source: the schema reference became the
-- language reference, and pricing an expression became previewing a source.
SELECT "agent_rename_tool"('formula_describe_schema', 'formula_describe_language');
SELECT "agent_rename_tool"('formula_test_expression', 'formula_preview');
