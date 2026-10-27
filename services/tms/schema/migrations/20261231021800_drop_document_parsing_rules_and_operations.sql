-- Tenant parsing rules and the document operations console are gone: extraction runs on its
-- built-in parsers only, and the grants and policies that named either resource go with them.
DROP TABLE IF EXISTS "document_parsing_rule_fixtures";

ALTER TABLE IF EXISTS "document_parsing_rule_sets"
    DROP CONSTRAINT IF EXISTS "fk_document_parsing_rule_sets_published_version";

DROP TABLE IF EXISTS "document_parsing_rule_versions";

DROP TABLE IF EXISTS "document_parsing_rule_sets";

DELETE FROM "resource_permissions"
WHERE "resource" IN ('document_parsing_rule', 'document_operation');

DELETE FROM "api_key_permissions"
WHERE "resource" IN ('document_parsing_rule', 'document_operation');

DELETE FROM "access_policies"
WHERE "resource" IN ('document_parsing_rule', 'document_operation');
