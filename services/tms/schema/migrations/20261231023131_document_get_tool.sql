-- The document read is document_get, as every other kind's read is <kind>_get.
-- The command palette's get_record finds a kind's read by that name, so it could
-- not open a document while the read was called document_get_summary.
SELECT "agent_rename_tool"('document_get_summary', 'document_get');
