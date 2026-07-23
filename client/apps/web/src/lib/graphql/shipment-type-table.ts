import {
  ShipmentTypeTableDocument,
  type ShipmentTypeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { ShipmentType } from "@/types/shipment-type";

export const shipmentTypeTableGraphQLConfig = defineDataTableGraphQLConfig<
  ShipmentType,
  ShipmentTypeTableQueryVariables
>({
  document: ShipmentTypeTableDocument,
  operationName: "ShipmentTypeTable",
  connectionKey: "shipmentTypes",
});
