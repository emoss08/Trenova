import {
  RecurringShipmentTableDocument,
  type RecurringShipmentTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { RecurringShipment } from "@/types/recurring-shipment";

export const recurringShipmentTableGraphQLConfig = defineDataTableGraphQLConfig<
  RecurringShipment,
  RecurringShipmentTableQueryVariables
>({
  document: RecurringShipmentTableDocument,
  operationName: "RecurringShipmentTable",
  connectionKey: "recurringShipments",
});
