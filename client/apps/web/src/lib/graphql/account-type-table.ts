import {
  AccountTypeTableDocument,
  type AccountTypeTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { AccountType } from "@/types/account-type";

export const accountTypeTableGraphQLConfig = defineDataTableGraphQLConfig<
  AccountType,
  AccountTypeTableQueryVariables
>({
  document: AccountTypeTableDocument,
  operationName: "AccountTypeTable",
  connectionKey: "accountTypes",
});
