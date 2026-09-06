import { ShipmentTypeTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const shipmentTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ShipmentTypeTableDocument,
  operationName: "ShipmentTypeTable",
  connectionKey: "shipmentTypes",
});

export type ShipmentTypeRow = DataTableConfigRow<typeof shipmentTypeTableGraphQLConfig>;
