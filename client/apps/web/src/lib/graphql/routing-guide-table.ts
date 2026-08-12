import {
  MatchRoutingGuideDocument,
  RoutingGuideOptionsDocument,
  RoutingGuideTableDocument,
  type MatchRoutingGuideInput,
  type MatchRoutingGuideQuery,
  type RoutingGuideOptionsQuery,
  type RoutingGuideTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { RoutingGuide } from "@trenova/shared/types/routing-guide";

export const routingGuideTableGraphQLConfig = defineDataTableGraphQLConfig<
  RoutingGuide,
  RoutingGuideTableQueryVariables
>({
  document: RoutingGuideTableDocument,
  operationName: "RoutingGuideTable",
  connectionKey: "routingGuides",
});

export type MatchedRoutingGuide = NonNullable<MatchRoutingGuideQuery["matchRoutingGuide"]>;
export type RoutingGuideOption = RoutingGuideOptionsQuery["routingGuides"]["edges"][number]["node"];

export async function matchRoutingGuideGraphQL(
  input: MatchRoutingGuideInput,
): Promise<MatchedRoutingGuide | null> {
  const data = await requestGraphQL({
    document: MatchRoutingGuideDocument,
    operationName: "MatchRoutingGuide",
    variables: { input },
  });
  return data.matchRoutingGuide ?? null;
}

export async function getRoutingGuideOptionsGraphQL(query: string): Promise<RoutingGuideOption[]> {
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
  });
  return data.routingGuides.edges.map((edge) => edge.node);
}
