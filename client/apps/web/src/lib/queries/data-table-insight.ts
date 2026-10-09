import {
  DataTableAggregatesDocument,
  DataTableFacetsDocument,
  DataTableInsightFieldsDocument,
  type DataTableAggregateInput,
  type DataTableFacetInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dataTableInsight = createQueryKeys("dataTableInsight", {
  fields: (resource: string) => ({
    queryKey: [resource],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: DataTableInsightFieldsDocument,
        operationName: "DataTableInsightFields",
        variables: { resource },
        signal,
      }),
  }),
  facets: (input: DataTableFacetInput) => ({
    queryKey: [input],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: DataTableFacetsDocument,
        operationName: "DataTableFacets",
        variables: { input },
        signal,
      }),
  }),
  /** `version` moves when the table's rows are refetched, so the totals follow them. */
  aggregates: (input: DataTableAggregateInput, version = 0) => ({
    queryKey: [input, version],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: DataTableAggregatesDocument,
        operationName: "DataTableAggregates",
        variables: { input },
        signal,
      }),
  }),
});
