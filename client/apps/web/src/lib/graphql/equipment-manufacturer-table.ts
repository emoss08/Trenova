import { EquipmentManufacturerTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const equipmentManufacturerTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: EquipmentManufacturerTableDocument,
  operationName: "EquipmentManufacturerTable",
  connectionKey: "equipmentManufacturers",
});
