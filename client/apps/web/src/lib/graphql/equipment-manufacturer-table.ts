import {
  EquipmentManufacturerTableDocument,
  type EquipmentManufacturerTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";

export const equipmentManufacturerTableGraphQLConfig = defineDataTableGraphQLConfig<
  EquipmentManufacturer,
  EquipmentManufacturerTableQueryVariables
>({
  document: EquipmentManufacturerTableDocument,
  operationName: "EquipmentManufacturerTable",
  connectionKey: "equipmentManufacturers",
});
