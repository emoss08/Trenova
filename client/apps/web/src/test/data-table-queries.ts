import type { QueryClient } from "@tanstack/react-query";

/**
 * The query factories a `DataTable` reads besides its own rows, answered with
 * empty results. Spread into a test's `@/lib/queries` mock so a table renders
 * without a server.
 */
export const dataTableQueryMocks = {
  bulkEdit: {
    fields: () => ({ queryKey: ["bulkEdit-fields"], queryFn: () => [] }),
    preview: () => ({
      queryKey: ["bulkEdit-preview"],
      queryFn: () => ({ count: 0, tooMany: false }),
    }),
    job: () => ({ queryKey: ["bulkEdit-job"], queryFn: () => null }),
  },
  tableLayout: {
    mine: () => ({ queryKey: ["tableLayout-mine"], queryFn: () => ({ myTableLayout: null }) }),
  },
  dataTableInsight: {
    fields: () => ({
      queryKey: ["dataTableInsight-fields"],
      queryFn: () => ({ dataTableInsightFields: [] }),
    }),
    aggregates: () => ({
      queryKey: ["dataTableInsight-aggregates"],
      queryFn: () => ({ dataTableAggregates: { count: 0, values: [] } }),
    }),
    facets: () => ({
      queryKey: ["dataTableInsight-facets"],
      queryFn: () => ({ dataTableFacets: { field: "", total: 0, buckets: [] } }),
    }),
  },
  tableConfiguration: {
    default: () => ({ queryKey: ["tableConfig-default"], queryFn: () => null }),
    all: () => ({
      queryKey: ["tableConfig-all"],
      queryFn: () => ({ results: [], count: 0 }),
    }),
  },
};

/**
 * Answers, before the first render, the two questions a table waits on before it
 * draws rows: the person's saved layout (none) and the table's default view (none).
 * A returning visit has both cached, so the table draws its rows at once.
 */
export function seedDataTableQueries(client: QueryClient): QueryClient {
  client.setQueryData(["tableLayout-mine"], { myTableLayout: null });
  client.setQueryData(["tableConfig-default"], null);
  return client;
}
