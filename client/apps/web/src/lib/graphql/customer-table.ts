import {
  CustomerTableDocument,
  type CustomerTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Customer } from "@trenova/shared/types/customer";

export const customerTableGraphQLConfig = defineDataTableGraphQLConfig<
  Customer,
  CustomerTableQueryVariables
>({
  document: CustomerTableDocument,
  operationName: "CustomerTable",
  connectionKey: "customers",
});
