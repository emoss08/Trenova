import {
  ServiceTypeTableDocument,
  type ServiceTypeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { ServiceType } from "@/types/service-type";

export const serviceTypeTableGraphQLConfig = defineDataTableGraphQLConfig<
  ServiceType,
  ServiceTypeTableQueryVariables
>({
  document: ServiceTypeTableDocument,
  operationName: "ServiceTypeTable",
  connectionKey: "serviceTypes",
});
