-- Two reads of the same records are one read, and an agent exception raised for review is the one exception tool: worker_list finds a worker by name or code, worker_list_credentials lists one worker's credentials or what falls due across the organization, inbox_list_inbound_messages ranks mail by a query, accounting_list_sync_records reads sync state by document ids, and agent_raise_exception takes evidence.
SELECT "agent_rename_tool"('accounting_get_sync_state', 'accounting_list_sync_records');
SELECT "agent_rename_tool"('agent_request_review', 'agent_raise_exception');
SELECT "agent_rename_tool"('inbox_search_inbound_messages', 'inbox_list_inbound_messages');
SELECT "agent_rename_tool"('worker_list_expiring_credentials', 'worker_list_credentials');
SELECT "agent_rename_tool"('worker_search', 'worker_list');
