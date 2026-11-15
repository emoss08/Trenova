import {
  EquipmentManufacturerTableDocument,
} from "@trenova/graphql/generated/documents/equipment-manufacturer/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const equipmentManufacturerTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: EquipmentManufacturerTableDocument,
  operationName: "EquipmentManufacturerTable",
  connectionKey: "equipmentManufacturers",
});
