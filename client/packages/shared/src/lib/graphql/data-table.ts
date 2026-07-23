import type { DataTableGraphQLConfig } from "@trenova/shared/types/data-table";

export const DATA_TABLE_CONNECTION_VARIABLES = `
  $input: DataTableConnectionInput!
`;

export const DATA_TABLE_CONNECTION_ARGUMENTS = `
  input: $input
`;

export const DATA_TABLE_PAGE_INFO_FRAGMENT = `
  fragment DataTablePageInfoFields on PageInfo {
    hasNextPage
    endCursor
  }
`;

export function defineDataTableGraphQLConfig<
  TData extends Record<string, unknown>,
  TVariables extends Record<string, unknown> = Record<string, unknown>,
>(config: DataTableGraphQLConfig<TData, TVariables>): DataTableGraphQLConfig<TData, TVariables> {
  return config;
}
