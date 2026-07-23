import {
  SelectOptionsDocument,
  type SelectOptionResource,
  type SelectOptionsInput,
  type SelectOptionsQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";

export type SelectOptionMeta = Record<string, unknown>;

export type SelectOption = {
  id: string;
  label: string;
  description: string | null;
  meta: SelectOptionMeta | null;
};

type FetchGraphQLSelectOptionsParams = {
  resource: SelectOptionResource;
  query?: string;
  page?: number;
  initialLimit?: number;
  filters?: Record<string, unknown>;
  ids?: string[];
};

export type GraphQLSelectOptionsConfig = {
  resource: SelectOptionResource;
  filters?: Record<string, unknown>;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function normalizeSelectOption(
  node: SelectOptionsQuery["selectOptions"]["edges"][number]["node"],
): SelectOption {
  return {
    id: String(node.id),
    label: node.label,
    description: node.description ?? null,
    meta: isRecord(node.meta) ? node.meta : null,
  };
}

function normalizeSelectOptionConnection(
  data: SelectOptionsQuery,
  offset: number,
  limit: number,
): GenericLimitOffsetResponse<SelectOption> {
  const connection = data.selectOptions;
  const results = connection.edges.map((edge) => normalizeSelectOption(edge.node));
  const count = connection.totalCount ?? results.length;
  const nextOffset = offset + limit;
  const previousOffset = Math.max(offset - limit, 0);

  return {
    results,
    count,
    next: connection.pageInfo.hasNextPage
      ? `graphql-select-options://${nextOffset.toString()}`
      : null,
    prev: offset > 0 ? `graphql-select-options://${previousOffset.toString()}` : null,
    pageInfo: {
      mode: "offset",
      hasNextPage: connection.pageInfo.hasNextPage,
      endCursor: connection.pageInfo.endCursor,
      totalCount: count,
    },
  };
}

export function selectOptionFiltersFromSearchParams(
  extraSearchParams?: Record<string, string | string[]>,
): Record<string, unknown> | undefined {
  if (!extraSearchParams) {
    return undefined;
  }

  const filters: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(extraSearchParams)) {
    filters[key] = value;
  }

  return Object.keys(filters).length > 0 ? filters : undefined;
}

export async function fetchGraphQLSelectOptions({
  resource,
  query,
  page = 1,
  initialLimit = 20,
  filters,
  ids,
}: FetchGraphQLSelectOptionsParams): Promise<GenericLimitOffsetResponse<SelectOption>> {
  const limit = initialLimit;
  const offset = ids?.length ? 0 : (page - 1) * limit;
  const input: SelectOptionsInput = {
    resource,
    first: limit,
    offset,
  };

  if (query) {
    input.query = query;
  }
  if (filters && Object.keys(filters).length > 0) {
    input.filters = filters;
  }
  if (ids?.length) {
    input.ids = ids;
  }

  const data = await requestGraphQL({
    document: SelectOptionsDocument,
    operationName: "SelectOptions",
    variables: { input },
  });

  return normalizeSelectOptionConnection(data, offset, limit);
}

export async function fetchGraphQLSelectedOption(
  resource: SelectOptionResource,
  id: string,
  filters?: Record<string, unknown>,
): Promise<SelectOption | null> {
  const response = await fetchGraphQLSelectOptions({
    resource,
    ids: [id],
    initialLimit: 1,
    filters,
  });

  return response.results[0] ?? null;
}
