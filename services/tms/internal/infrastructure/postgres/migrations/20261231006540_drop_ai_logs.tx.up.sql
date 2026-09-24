-- ai_logs was write-only: nothing read it, and it kept previews of prompts and
-- replies, including document text, for as long as the organization existed.
-- Every call it recorded is now an ai_usage_records row that names the feature
-- and the record the call was for, with the provider, tokens, cost and latency
-- ai_logs never had. The rows are not carried over; the table goes.
DROP TABLE IF EXISTS "ai_logs";

--bun:split
DROP FUNCTION IF EXISTS prevent_ai_logs_modification();

--bun:split
DROP TYPE IF EXISTS "operation_enum";
