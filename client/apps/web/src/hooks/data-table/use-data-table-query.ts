import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
} from "@trenova/shared/types/data-table";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { useQuery } from "@tanstack/react-query";
import type { PaginationState } from "@tanstack/react-table";

export type { DataTableQueryOptions } from "@trenova/shared/types/data-table";

type GraphQLConnection<TNode> = {
  edges?: Array<{ node: TNode }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
};

type DataTableGraphQLVariables = Record<string, unknown> & {
  includeTotalCount: boolean;
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
  graphql: DataTableGraphQLSource<TData>;
  signal?: AbortSignal;
};

function resolveExtraVariables<TData extends Record<string, unknown>>(
  config: DataTableGraphQLSource<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
): Record<string, unknown> {
  if (!config.extraVariables) {
    return {};
  }

  if (typeof config.extraVariables === "function") {
    return config.extraVariables({ pageSize, options });
  }

  return config.extraVariables;
}

function resolveInputExtraVariables<TData extends Record<string, unknown>>(
  config: DataTableGraphQLSource<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
): Record<string, unknown> {
  if (!config.inputExtraVariables) {
    return {};
  }

  if (typeof config.inputExtraVariables === "function") {
    return config.inputExtraVariables({ pageSize, options });
  }

  return config.inputExtraVariables;
}

function buildGraphQLVariables<TData extends Record<string, unknown>>(
  pageSize: number,
  config: DataTableGraphQLSource<TData>,
  options?: DataTableQueryOptions,
): DataTableGraphQLVariables {
  return {
    includeTotalCount: !options?.cursor,
    input: {
      first: pageSize,
      after: options?.cursor || undefined,
      query: options?.query || undefined,
      fieldFilters: options?.fieldFilters ?? [],
      filterGroups: options?.filterGroups ?? [],
      sort: options?.sort ?? [],
      ...resolveInputExtraVariables(config, pageSize, options),
    },
    ...resolveExtraVariables(config, pageSize, options),
  };
}

export async function fetchGraphQLData<TData extends Record<string, unknown>>(
  pageSize: number,
  config: DataTableGraphQLSource<TData>,
  options?: DataTableQueryOptions,
  requestOptions?: { signal?: AbortSignal },
): Promise<GenericLimitOffsetResponse<TData>> {
  const data = await requestGraphQL<
    Record<string, GraphQLConnection<unknown>>,
    DataTableGraphQLVariables
  >({
    document: config.document,
    operationName: config.operationName,
    variables: buildGraphQLVariables(pageSize, config, options),
    signal: requestOptions?.signal,
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
  signal,
}: FetchDataTablePageParams<TData>): Promise<GenericLimitOffsetResponse<TData>> {
  return fetchGraphQLData(pageSize, graphql, options, { signal });
}

export function buildDataTableQueryKey<TData extends Record<string, unknown>>(
  queryKey: string,
  graphql: DataTableGraphQLSource<TData>,
  pagination: PaginationState,
  options?: DataTableQueryOptions,
) {
  return [
    queryKey,
    pagination,
    options,
    {
      connectionKey: graphql.connectionKey,
      operationName: graphql.operationName,
      extraVariables: resolveExtraVariables(graphql, pagination.pageSize, options),
      inputExtraVariables: resolveInputExtraVariables(graphql, pagination.pageSize, options),
    },
  ] as const;
}

export function useDataTableQuery<TData extends Record<string, unknown>>(
  queryKey: string,
  graphql: DataTableGraphQLSource<TData>,
  pagination: PaginationState,
  options?: DataTableQueryOptions,
  enabled = true,
) {
  // oxlint-disable-next-line @tanstack/query/exhaustive-deps
  return useQuery<GenericLimitOffsetResponse<TData>, Error>({
    queryKey: buildDataTableQueryKey(queryKey, graphql, pagination, options),
    queryFn: async ({ signal }) =>
      fetchDataTablePage<TData>({
        pageSize: pagination.pageSize,
        options,
        graphql,
        signal,
      }),
    enabled,
  });
}
