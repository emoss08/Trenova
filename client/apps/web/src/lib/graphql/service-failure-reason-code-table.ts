import { ServiceFailureReasonCodeTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const serviceFailureReasonCodeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ServiceFailureReasonCodeTableDocument,
  operationName: "ServiceFailureReasonCodeTable",
  connectionKey: "serviceFailureReasonCodes",
});

export type ServiceFailureReasonCodeRow = DataTableConfigRow<
  typeof serviceFailureReasonCodeTableGraphQLConfig
>;
