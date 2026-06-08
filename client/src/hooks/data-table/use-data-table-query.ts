import { requestGraphQL } from "@/lib/graphql";
import type { DataTableGraphQLConfig, DataTableQueryOptions } from "@/types/data-table";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import type { PaginationState } from "@tanstack/react-table";

export type { DataTableQueryOptions } from "@/types/data-table";

type GraphQLConnection<TNode> = {
  edges?: Array<{ node: TNode }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
};

type DataTableGraphQLVariables = Record<string, unknown> & {
  input: {
    first: number;
    after?: string;
    query?: string;
    fieldFilters: DataTableQueryOptions["fieldFilters"];
    filterGroups: DataTableQueryOptions["filterGroups"];
    sort: DataTableQueryOptions["sort"];
  };
};

type FetchDataTablePageParams<TData extends Record<string, unknown>> = {
  pageSize: number;
  options?: DataTableQueryOptions;
  graphql: DataTableGraphQLConfig<TData>;
};

function resolveExtraVariables<TData extends Record<string, unknown>>(
  config: DataTableGraphQLConfig<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
): Record<string, unknown> {
  if (!config.extraVariables) {
    return {};
  }

  if (typeof config.extraVariables === "function") {
    return config.extraVariables({ pageSize, options }) as Record<string, unknown>;
  }

  return config.extraVariables as Record<string, unknown>;
}

function buildGraphQLVariables<TData extends Record<string, unknown>>(
  pageSize: number,
  config: DataTableGraphQLConfig<TData>,
  options?: DataTableQueryOptions,
): DataTableGraphQLVariables {
  return {
    input: {
      first: pageSize,
      after: options?.cursor || undefined,
      query: options?.query || undefined,
      fieldFilters: options?.fieldFilters ?? [],
      filterGroups: options?.filterGroups ?? [],
      sort: options?.sort ?? [],
    },
    ...resolveExtraVariables(config, pageSize, options),
  };
}

export async function fetchGraphQLData<TData extends Record<string, unknown>>(
  pageSize: number,
  config: DataTableGraphQLConfig<TData>,
  options?: DataTableQueryOptions,
): Promise<GenericLimitOffsetResponse<TData>> {
  const data = await requestGraphQL<
    Record<string, GraphQLConnection<unknown>>,
    DataTableGraphQLVariables
  >({
    document: config.document,
    operationName: config.operationName,
    variables: buildGraphQLVariables(pageSize, config, options),
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
    pageInfo: {
      mode: "cursor",
      hasNextPage: connection.pageInfo?.hasNextPage ?? false,
      endCursor: connection.pageInfo?.endCursor ?? null,
      totalCount,
    },
  };
}

export async function fetchDataTablePage<TData extends Record<string, unknown>>({
  pageSize,
  options,
  graphql,
}: FetchDataTablePageParams<TData>): Promise<GenericLimitOffsetResponse<TData>> {
  return fetchGraphQLData(pageSize, graphql, options);
}

export function useDataTableQuery<TData extends Record<string, unknown>>(
  queryKey: string,
  graphql: DataTableGraphQLConfig<TData>,
  pagination: PaginationState,
  options?: DataTableQueryOptions,
  enabled = true,
) {
  const extraVariables = resolveExtraVariables(graphql, pagination.pageSize, options);

  return useQuery<GenericLimitOffsetResponse<TData>, Error>({
    queryKey: [
      queryKey,
      pagination,
      options,
      {
        connectionKey: graphql.connectionKey,
        document: graphql.document.toString(),
        graphql,
        operationName: graphql.operationName,
        extraVariables,
      },
    ],
    queryFn: async () =>
      fetchDataTablePage<TData>({
        pageSize: pagination.pageSize,
        options,
        graphql,
      }),
    enabled,
  });
}
