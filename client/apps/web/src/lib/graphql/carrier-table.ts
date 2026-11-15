import { CarrierTableDocument } from "@trenova/graphql/generated/documents/carrier/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const carrierTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: CarrierTableDocument,
  operationName: "CarrierTable",
  connectionKey: "carriers",
});

export type CarrierRow = DataTableConfigRow<typeof carrierTableGraphQLConfig>;
