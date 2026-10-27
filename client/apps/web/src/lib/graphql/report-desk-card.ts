import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskReportRunCardDocument,
  ReportRunFieldsFragmentDoc,
  type DeskReportRunCardQuery,
  type ReportRunFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RawCard = DeskReportRunCardQuery["reportRunDeskCard"];

export type ReportRunDeskParameter = RawCard["parameters"][number];
export type ReportRunDeskSampleColumn = NonNullable<RawCard["sample"]>["columns"][number];
export type ReportRunDeskChart = NonNullable<RawCard["chart"]>;
export type ReportRunDeskSort = RawCard["sort"][number];

/** A run's stored sample: the columns, its first rows and grand totals, positional. */
export type ReportRunDeskSample = {
  columns: ReportRunDeskSampleColumn[];
  rows: unknown[][];
  totals: unknown[] | null;
  more: boolean;
};

/** The previous comparable run's grand totals, by column id. */
export type ReportRunDeskPrevious = {
  runId: string;
  completedAt: number;
  rowCount: number;
  totals: Map<string, unknown>;
};

/** A report run as the Desk card draws it. */
export type ReportRunDeskCard = Omit<RawCard, "run" | "sample" | "previous"> & {
  run: ReportRunFieldsFragment;
  sample: ReportRunDeskSample | null;
  previous: ReportRunDeskPrevious | null;
};

function rowsOf(value: unknown): unknown[][] {
  return Array.isArray(value) ? value.filter((row): row is unknown[] => Array.isArray(row)) : [];
}

export async function fetchReportRunDeskCard(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<ReportRunDeskCard> {
  const data = await requestGraphQL({
    document: DeskReportRunCardDocument,
    operationName: "DeskReportRunCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.reportRunDeskCard;
  const previousTotals =
    card.previous && Array.isArray(card.previous.totals) ? card.previous.totals : [];

  return {
    ...card,
    run: getFragmentData(ReportRunFieldsFragmentDoc, card.run),
    sample: card.sample
      ? {
          columns: card.sample.columns,
          rows: rowsOf(card.sample.rows),
          totals: Array.isArray(card.sample.totals) ? card.sample.totals : null,
          more: card.sample.more,
        }
      : null,
    previous: card.previous
      ? {
          runId: card.previous.runId,
          completedAt: card.previous.completedAt,
          rowCount: card.previous.rowCount,
          totals: new Map(
            card.previous.columnIds.map((columnId, index) => [columnId, previousTotals[index]]),
          ),
        }
      : null,
  };
}
