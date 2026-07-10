import {
  ServiceFailureTableDocument,
  type ServiceFailureTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { ServiceFailure } from "@/types/service-failure";

export const serviceFailureTableGraphQLConfig = defineDataTableGraphQLConfig<
  ServiceFailure,
  ServiceFailureTableQueryVariables
>({
  document: ServiceFailureTableDocument,
  operationName: "ServiceFailureTable",
  connectionKey: "serviceFailures",
});
