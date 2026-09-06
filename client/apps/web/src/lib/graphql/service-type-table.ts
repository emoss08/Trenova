import { ServiceTypeTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const serviceTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ServiceTypeTableDocument,
  operationName: "ServiceTypeTable",
  connectionKey: "serviceTypes",
});

export type ServiceTypeRow = DataTableConfigRow<typeof serviceTypeTableGraphQLConfig>;
