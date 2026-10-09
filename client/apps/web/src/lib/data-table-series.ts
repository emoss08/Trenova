import { DataTableSeriesDocument } from "@trenova/graphql/generated/graphql";
import { createBatchLoader, type BatchLoader } from "@trenova/shared/lib/batch-loader";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

/** A chart drawn in every row: one table's rows, counted or totalled per period, for each row. */
export type DataTableSeriesSpec = {
  /** The table counted, such as shipment. */
  resource: string;
  /** The field on that table naming the row a chart belongs to, such as customerId. */
  groupField: string;
  /** The date that places a counted row in a period. */
  dateField: string;
  /** A field to total instead of counting rows. */
  valueField?: string;
  interval?: "week" | "month";
  periods?: number;
};

export type DataTableSeriesValue = {
  periodStarts: readonly number[];
  points: readonly number[];
};

const DEFAULT_PERIODS = 12;

export function seriesSpecKey(spec: DataTableSeriesSpec): string {
  return [
    spec.resource,
    spec.groupField,
    spec.dateField,
    spec.valueField ?? "",
    spec.interval ?? "week",
    spec.periods ?? DEFAULT_PERIODS,
  ].join("|");
}

const loaders = new Map<string, BatchLoader<string, DataTableSeriesValue>>();

/**
 * The loader for one chart. Every row asking in the same tick shares one request, at
 * most a page of rows at a time, and a row with nothing in the window charts as zeros.
 */
export function seriesLoader(spec: DataTableSeriesSpec): BatchLoader<string, DataTableSeriesValue> {
  const key = seriesSpecKey(spec);
  const existing = loaders.get(key);
  if (existing) return existing;

  const loader = createBatchLoader<string, DataTableSeriesValue>(
    async (ids) => {
      const data = await requestGraphQL({
        document: DataTableSeriesDocument,
        operationName: "DataTableSeries",
        variables: {
          input: {
            resource: spec.resource,
            groupField: spec.groupField,
            groupValues: [...ids],
            dateField: spec.dateField,
            valueField: spec.valueField,
            interval: spec.interval === "month" ? "MONTH" : "WEEK",
            periods: spec.periods ?? DEFAULT_PERIODS,
            timezone: resolveUserTimezone(),
          },
        },
      });
      const { periodStarts, groups } = data.dataTableSeries;
      const answers = new Map<string, DataTableSeriesValue>();
      for (const group of groups) {
        answers.set(group.value, { periodStarts, points: group.points.map(Number) });
      }
      return answers;
    },
    { maxBatchSize: 200 },
  );
  loaders.set(key, loader);
  return loader;
}
