import { RecurringShipmentTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const recurringShipmentTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RecurringShipmentTableDocument,
  operationName: "RecurringShipmentTable",
  connectionKey: "recurringShipments",
});

export type RecurringShipmentRow = DataTableConfigRow<typeof recurringShipmentTableGraphQLConfig>;
