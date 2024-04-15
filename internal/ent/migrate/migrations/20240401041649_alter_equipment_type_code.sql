-- Modify "equipment_types" table
ALTER TABLE "equipment_types" DROP COLUMN "name", ADD COLUMN "code" character varying(10) NOT NULL;
-- Create index "equipmenttype_code_organization_id" to table: "equipment_types"
CREATE UNIQUE INDEX "equipmenttype_code_organization_id" ON "equipment_types" ("code", "organization_id");
