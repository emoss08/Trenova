import {
  MatchRoutingGuideDocument,
  RoutingGuideOptionsDocument,
  RoutingGuideTableDocument,
  type MatchRoutingGuideInput,
  type MatchRoutingGuideQuery,
  type RoutingGuideOptionsQuery,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export const routingGuideTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: RoutingGuideTableDocument,
  operationName: "RoutingGuideTable",
  connectionKey: "routingGuides",
});

export type RoutingGuideRow = DataTableConfigRow<typeof routingGuideTableGraphQLConfig>;

export type MatchedRoutingGuide = NonNullable<MatchRoutingGuideQuery["matchRoutingGuide"]>;
export type RoutingGuideOption = RoutingGuideOptionsQuery["routingGuides"]["edges"][number]["node"];

export async function matchRoutingGuideGraphQL(
  input: MatchRoutingGuideInput,
  options?: { signal?: AbortSignal },
): Promise<MatchedRoutingGuide | null> {
  const data = await requestGraphQL({
    document: MatchRoutingGuideDocument,
    operationName: "MatchRoutingGuide",
    variables: { input },
    signal: options?.signal,
  });
  return data.matchRoutingGuide ?? null;
}

export async function getRoutingGuideOptionsGraphQL(
  query: string,
  options?: { signal?: AbortSignal },
): Promise<RoutingGuideOption[]> {
  const data = await requestGraphQL({
    document: RoutingGuideOptionsDocument,
    operationName: "RoutingGuideOptions",
    variables: {
      input: {
        first: 50,
        query: query || undefined,
        fieldFilters: [{ field: "status", operator: "eq", value: "Active" }],
      },
    },
    signal: options?.signal,
  });
  return data.routingGuides.edges.map((edge) => edge.node);
}
