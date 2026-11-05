-- Global search has no owner check, so a file attached to a conversation is no longer projected
-- for it. The rows already there leave with this; the sink follows the delete.
DELETE FROM public.document_search_projections
WHERE resource_type = 'assistant_thread';
