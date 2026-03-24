DROP TRIGGER IF EXISTS generated_documents_search_update ON "generated_documents";

--bun:split
DROP FUNCTION IF EXISTS generated_documents_search_trigger();

--bun:split
DROP TABLE IF EXISTS "generated_documents";

--bun:split
DROP TRIGGER IF EXISTS document_templates_search_update ON "document_templates";

--bun:split
DROP FUNCTION IF EXISTS document_templates_search_trigger();

--bun:split
DROP TABLE IF EXISTS "document_templates";

--bun:split
DROP TYPE IF EXISTS delivery_method_enum;

--bun:split
DROP TYPE IF EXISTS generation_status_enum;

--bun:split
DROP TYPE IF EXISTS orientation_enum;

--bun:split
DROP TYPE IF EXISTS page_size_enum;

--bun:split
DROP TYPE IF EXISTS template_status_enum;
