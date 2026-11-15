-- The bytes every organization stores, kept by triggers on documents so a managed instance reads
-- its storage level, and checks an upload against it, from one row per organization rather than
-- by adding up every document. Statement triggers see every writer: an upload, a bulk delete, a
-- lineage delete and a cascade from a deleted organization.
CREATE TABLE IF NOT EXISTS "document_storage_totals"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "bytes" bigint NOT NULL DEFAULT 0,
    CONSTRAINT "pk_document_storage_totals" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_storage_totals_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_storage_totals_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "document_storage_totals" IS 'The bytes of every stored document, per organization, kept by triggers on documents';

COMMENT ON COLUMN "document_storage_totals"."bytes" IS 'The sum of file_size over the organization''s documents';

CREATE OR REPLACE FUNCTION document_storage_totals_apply() RETURNS trigger
LANGUAGE plpgsql AS $fn$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO "document_storage_totals" ("organization_id", "business_unit_id", "bytes")
        SELECT n."organization_id", n."business_unit_id", sum(n."file_size")
          FROM document_storage_new n
         GROUP BY 1, 2
        ON CONFLICT ("organization_id", "business_unit_id")
        DO UPDATE SET "bytes" = "document_storage_totals"."bytes" + EXCLUDED."bytes";
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE "document_storage_totals" t
           SET "bytes" = t."bytes" - o."bytes"
          FROM (
                SELECT "organization_id", "business_unit_id", sum("file_size") AS "bytes"
                  FROM document_storage_old
                 GROUP BY 1, 2
               ) o
         WHERE t."organization_id" = o."organization_id"
           AND t."business_unit_id" = o."business_unit_id";
    ELSE
        INSERT INTO "document_storage_totals" ("organization_id", "business_unit_id", "bytes")
        SELECT c."organization_id", c."business_unit_id", sum(c."delta")
          FROM (
                SELECT "organization_id", "business_unit_id", "file_size" AS "delta"
                  FROM document_storage_new
                UNION ALL
                SELECT "organization_id", "business_unit_id", -"file_size"
                  FROM document_storage_old
               ) c
         GROUP BY 1, 2
        HAVING sum(c."delta") <> 0
        ON CONFLICT ("organization_id", "business_unit_id")
        DO UPDATE SET "bytes" = "document_storage_totals"."bytes" + EXCLUDED."bytes";
    END IF;
    RETURN NULL;
END
$fn$;

CREATE OR REPLACE TRIGGER "trg_documents_storage_totals_insert"
    AFTER INSERT ON "documents"
    REFERENCING NEW TABLE AS document_storage_new
    FOR EACH STATEMENT EXECUTE FUNCTION document_storage_totals_apply();

CREATE OR REPLACE TRIGGER "trg_documents_storage_totals_update"
    AFTER UPDATE ON "documents"
    REFERENCING OLD TABLE AS document_storage_old NEW TABLE AS document_storage_new
    FOR EACH STATEMENT EXECUTE FUNCTION document_storage_totals_apply();

CREATE OR REPLACE TRIGGER "trg_documents_storage_totals_delete"
    AFTER DELETE ON "documents"
    REFERENCING OLD TABLE AS document_storage_old
    FOR EACH STATEMENT EXECUTE FUNCTION document_storage_totals_apply();

INSERT INTO "document_storage_totals" ("organization_id", "business_unit_id", "bytes")
SELECT "organization_id", "business_unit_id", sum("file_size")
  FROM "documents"
 GROUP BY 1, 2
ON CONFLICT ("organization_id", "business_unit_id") DO UPDATE SET "bytes" = EXCLUDED."bytes";
