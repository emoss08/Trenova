ALTER TABLE "document_parsing_rule_sets"
    DROP CONSTRAINT IF EXISTS "fk_document_parsing_rule_sets_published_version";

DROP TABLE IF EXISTS "document_parsing_rule_fixtures";
DROP TABLE IF EXISTS "document_parsing_rule_versions";
DROP TABLE IF EXISTS "document_parsing_rule_sets";
