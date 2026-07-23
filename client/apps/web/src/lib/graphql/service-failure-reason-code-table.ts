import {
  ServiceFailureReasonCodeTableDocument,
  type ServiceFailureReasonCodeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";

export const serviceFailureReasonCodeTableGraphQLConfig = defineDataTableGraphQLConfig<
  ServiceFailureReasonCode,
  ServiceFailureReasonCodeTableQueryVariables
>({
  document: ServiceFailureReasonCodeTableDocument,
  operationName: "ServiceFailureReasonCodeTable",
  connectionKey: "serviceFailureReasonCodes",
});
