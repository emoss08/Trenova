import type { ResultOf } from "@graphql-typed-document-node/core";
import type {
  ConnectionKeys,
  DataTableGraphQLConfig,
  DataTableRow,
} from "@trenova/shared/types/data-table";
import type { TypedGraphQLDocument } from "@trenova/shared/types/graphql";

export function defineDataTableGraphQLConfig<
  TDocument extends TypedGraphQLDocument<unknown, never>,
  const K extends ConnectionKeys<ResultOf<TDocument>>,
  TData = DataTableRow<TDocument, K>,
>(
  config: DataTableGraphQLConfig<TDocument, K, TData>,
): DataTableGraphQLConfig<TDocument, K, TData> {
  return config;
}
