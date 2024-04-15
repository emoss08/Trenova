-- Modify "document_classifications" table
ALTER TABLE "document_classifications" DROP COLUMN "name", ADD COLUMN "code" character varying(10) NOT NULL, ADD COLUMN "color" character varying NULL;
-- Create index "documentclassification_code_organization_id" to table: "document_classifications"
CREATE UNIQUE INDEX "documentclassification_code_organization_id" ON "document_classifications" ("code", "organization_id");
