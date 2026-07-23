import {
  ApiKeyTableDocument,
  type ApiKeyTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { ApiKey } from "@/types/api-key";

export const apiKeyTableGraphQLConfig = defineDataTableGraphQLConfig<
  ApiKey,
  ApiKeyTableQueryVariables
>({
  document: ApiKeyTableDocument,
  operationName: "ApiKeyTable",
  connectionKey: "apiKeys",
});
