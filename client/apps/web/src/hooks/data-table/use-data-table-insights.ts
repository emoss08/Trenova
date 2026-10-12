import { queries } from "@/lib/queries";
import type { DataTableAggregatesQuery } from "@trenova/graphql/generated/graphql";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { DataTableQueryOptions } from "@trenova/shared/types/data-table";
import { useMemo } from "react";

export type DataTableAggregateValues = DataTableAggregatesQuery["dataTableAggregates"]["values"][number];

export type DataTableTotals = {
  count: number;
  byField: ReadonlyMap<string, DataTableAggregateValues>;
};

type UseDataTableInsightsParams = {
  /** The table's permission resource; without one there are no counts or totals. */
  resource: string | undefined;
  /** The filters the table is showing, so counts and totals match its rows. */
  queryOptions: DataTableQueryOptions;
  /** The table's own list options beyond filters, such as a board's quick filters. */
  options: Record<string, unknown>;
  /** The fields the visible columns total. */
  aggregateFields: readonly string[];
  showTotals: boolean;
  /** When the page was last answered: totals refresh with it. */
  dataVersion: number;
};

const NO_FIELDS: ReadonlySet<string> = new Set();

/**
 * What the server can count and total for a table, and the totals of the visible
 * columns across every row the filters match. Totals refresh whenever the page does,
 * so a live update moves them too.
 */
export function useDataTableInsights({
  resource,
  queryOptions,
  options,
  aggregateFields,
  showTotals,
  dataVersion,
}: UseDataTableInsightsParams) {
  const fieldsQuery = useQuery({
    ...queries.dataTableInsight.fields(resource ?? ""),
    enabled: !!resource,
    staleTime: Infinity,
    retry: false,
  });

  const { facetable, summable } = useMemo(() => {
    const fields = fieldsQuery.data?.dataTableInsightFields;
    if (!fields) return { facetable: NO_FIELDS, summable: NO_FIELDS };
    return {
      facetable: new Set(fields.filter((field) => field.facetable).map((field) => field.name)),
      summable: new Set(fields.filter((field) => field.summable).map((field) => field.name)),
    };
  }, [fieldsQuery.data]);

  const totalled = useMemo(
    () => aggregateFields.filter((field) => summable.has(field)),
    [aggregateFields, summable],
  );

  const filter = useMemo(
    () => ({
      query: queryOptions.query || undefined,
      fieldFilters: queryOptions.fieldFilters ?? [],
      filterGroups: queryOptions.filterGroups ?? [],
    }),
    [queryOptions.query, queryOptions.fieldFilters, queryOptions.filterGroups],
  );

  const aggregatesQuery = useQuery({
    ...queries.dataTableInsight.aggregates(
      { resource: resource ?? "", fields: totalled, filter, options },
      dataVersion,
    ),
    enabled: !!resource && showTotals && totalled.length > 0,
    placeholderData: keepPreviousData,
    retry: false,
  });

  const totals = useMemo<DataTableTotals | undefined>(() => {
    const data = aggregatesQuery.data?.dataTableAggregates;
    if (!data) return undefined;
    return {
      count: data.count,
      byField: new Map(data.values.map((value) => [value.field, value])),
    };
  }, [aggregatesQuery.data]);

  return {
    facetable,
    summable,
    totals,
    totalsLoading: aggregatesQuery.isPending && aggregatesQuery.fetchStatus === "fetching",
    filter,
  };
}
