import {
  EquipmentManufacturerTableDocument,
  type EquipmentManufacturerTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";

export const equipmentManufacturerTableGraphQLConfig = defineDataTableGraphQLConfig<
  EquipmentManufacturer,
  EquipmentManufacturerTableQueryVariables
>({
  document: EquipmentManufacturerTableDocument,
  operationName: "EquipmentManufacturerTable",
  connectionKey: "equipmentManufacturers",
});
