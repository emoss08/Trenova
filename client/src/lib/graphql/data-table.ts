import type { DataTableGraphQLConfig } from "@/types/data-table";

export const DATA_TABLE_CONNECTION_VARIABLES = `
  $first: Int!
  $offset: Int
  $after: String
  $query: String
  $fieldFilters: [FieldFilterInput!]
  $filterGroups: [FilterGroupInput!]
  $sort: [SortFieldInput!]
`;

export const DATA_TABLE_CONNECTION_ARGUMENTS = `
  first: $first
  offset: $offset
  after: $after
  query: $query
  fieldFilters: $fieldFilters
  filterGroups: $filterGroups
  sort: $sort
`;

export const DATA_TABLE_PAGE_INFO_FRAGMENT = `
  fragment DataTablePageInfoFields on PageInfo {
    hasNextPage
    endCursor
  }
`;

export function defineDataTableGraphQLConfig<TData extends Record<string, unknown>>(
  config: DataTableGraphQLConfig<TData>,
): DataTableGraphQLConfig<TData> {
  return config;
}
