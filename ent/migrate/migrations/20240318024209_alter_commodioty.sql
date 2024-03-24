-- Modify "commodities" table
ALTER TABLE "commodities" DROP CONSTRAINT "commodities_hazardous_materials_commodities", ALTER COLUMN "hazardous_material_id" DROP NOT NULL, ADD CONSTRAINT "commodities_hazardous_materials_hazardous_material" FOREIGN KEY ("hazardous_material_id") REFERENCES "hazardous_materials" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
