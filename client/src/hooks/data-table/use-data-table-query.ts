import { API_BASE_URL } from "@/lib/constants";
import { requestGraphQL } from "@/lib/graphql";
import type {
  DataTableGraphQLConfig,
  FieldFilter,
  FilterGroup,
  SortField,
} from "@/types/data-table";
import type { API_ENDPOINTS, GenericLimitOffsetResponse } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import type { PaginationState } from "@tanstack/react-table";

export type DataTableQueryOptions = {
  query?: string;
  fieldFilters?: FieldFilter[];
  filterGroups?: FilterGroup[];
  sort?: SortField[];
  cursor?: string | null;
  extraSearchParams?: Record<string, unknown>;
};

type GraphQLConnection<TNode> = {
  edges?: Array<{ node: TNode }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
};

type FetchDataTablePageParams<TData extends Record<string, unknown>> = {
  link: API_ENDPOINTS;
  pageIndex: number;
  pageSize: number;
  options?: DataTableQueryOptions;
  graphql?: DataTableGraphQLConfig<TData>;
};

export async function fetchData<TData extends Record<string, unknown>>(
  link: string,
  pageIndex: number,
  pageSize: number,
  options?: DataTableQueryOptions,
): Promise<GenericLimitOffsetResponse<TData>> {
  const fetchURL = new URL(`${API_BASE_URL}${link}`, window.location.origin);
  fetchURL.searchParams.set("limit", pageSize.toString());
  fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());

  if (options?.query) {
    fetchURL.searchParams.set("query", options.query);
  }

  if (options?.fieldFilters && options.fieldFilters.length > 0) {
    fetchURL.searchParams.set(
      "fieldFilters",
      JSON.stringify(options.fieldFilters),
    );
  }

  if (options?.filterGroups && options.filterGroups.length > 0) {
    fetchURL.searchParams.set(
      "filterGroups",
      JSON.stringify(options.filterGroups),
    );
  }

  if (options?.sort && options.sort.length > 0) {
    fetchURL.searchParams.set("sort", JSON.stringify(options.sort));
  }

  if (options?.extraSearchParams) {
    Object.entries(options.extraSearchParams).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        if (typeof value === "string") {
          fetchURL.searchParams.set(key, value);
        } else if (typeof value === "boolean" || typeof value === "number") {
          fetchURL.searchParams.set(key, String(value));
        } else if (Array.isArray(value) || typeof value === "object") {
          fetchURL.searchParams.set(key, JSON.stringify(value));
        }
      }
    });
  }

  const response = await fetch(fetchURL.href, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Failed to fetch data from server");
  }

  return response.json();
}

export async function fetchGraphQLData<TData extends Record<string, unknown>>(
  pageSize: number,
  config: DataTableGraphQLConfig<TData>,
  options?: DataTableQueryOptions,
  pageIndex = 0,
): Promise<GenericLimitOffsetResponse<TData>> {
  const useOffsetPagination = (options?.sort?.length ?? 0) > 0;
  const variables = {
    ...config.variables,
    first: pageSize,
    offset: useOffsetPagination ? pageIndex * pageSize : undefined,
    after: useOffsetPagination ? undefined : options?.cursor || undefined,
    query: options?.query || undefined,
    fieldFilters: options?.fieldFilters ?? [],
    filterGroups: options?.filterGroups ?? [],
    sort: options?.sort ?? [],
  };
  const data = await requestGraphQL<Record<string, GraphQLConnection<unknown>>>({
    document: config.document,
    operationName: config.operationName,
    variables,
  });
  const connection = data[config.connectionKey];

  if (!connection) {
    throw new Error(`GraphQL response missing ${config.connectionKey} connection`);
  }

  const edges = connection.edges ?? [];
  const results = edges.map((edge) =>
    config.mapNode ? config.mapNode(edge.node) : (edge.node as TData),
  );
  const totalCount = connection.totalCount ?? null;

  return {
    results,
    count: totalCount ?? results.length,
    next: null,
    prev: null,
    pageInfo: useOffsetPagination
      ? undefined
      : {
          mode: "cursor",
          hasNextPage: connection.pageInfo?.hasNextPage ?? false,
          endCursor: connection.pageInfo?.endCursor ?? null,
          totalCount,
        },
  };
}

export async function fetchDataTablePage<TData extends Record<string, unknown>>({
  link,
  pageIndex,
  pageSize,
  options,
  graphql,
}: FetchDataTablePageParams<TData>): Promise<GenericLimitOffsetResponse<TData>> {
  if (graphql) {
    return fetchGraphQLData(pageSize, graphql, options, pageIndex);
  }

  return fetchData<TData>(link, pageIndex, pageSize, options);
}

export function useDataTableQuery<TData extends Record<string, unknown>>(
  queryKey: string,
  link: API_ENDPOINTS,
  pagination: PaginationState,
  options?: DataTableQueryOptions,
  graphql?: DataTableGraphQLConfig<TData>,
  enabled = true,
) {
  return useQuery<GenericLimitOffsetResponse<TData>, Error>({
    queryKey: [
      queryKey,
      link,
      pagination,
      options,
      graphql
        ? {
            connectionKey: graphql.connectionKey,
            document: graphql.document.toString(),
            operationName: graphql.operationName,
            variables: graphql.variables,
          }
        : null,
    ],
    queryFn: async () =>
      fetchDataTablePage<TData>({
        link,
        pageIndex: pagination.pageIndex,
        pageSize: pagination.pageSize,
        options,
        graphql,
      }),
    enabled,
    // structuralSharing: false,
  });
}
