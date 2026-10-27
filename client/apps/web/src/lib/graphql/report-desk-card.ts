import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskReportRunCardDocument,
  ReportPreviewFieldsFragmentDoc,
  ReportRunFieldsFragmentDoc,
  type DeskReportRunCardQuery,
  type ReportPreviewFieldsFragment,
  type ReportRunFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RawCard = DeskReportRunCardQuery["reportRunDeskCard"];

export type ReportRunDeskParameter = RawCard["parameters"][number];
export type ReportRunDeskSampleColumn = ReportPreviewFieldsFragment["columns"][number];
export type ReportRunDeskChart = NonNullable<RawCard["chart"]>;
export type ReportRunDeskSort = RawCard["sort"][number];

/** A run's stored sample: the columns, its first detail rows and grand totals, positional. */
export type ReportRunDeskSample = {
  columns: ReportRunDeskSampleColumn[];
  rows: unknown[][];
  totals: unknown[] | null;
  /** The run returned detail rows the sample left out. */
  truncated: boolean;
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

/** The detail rows of a sample: a subtotal row (level above 0) is not one of the result's rows. */
function detailRowsOf(sample: ReportPreviewFieldsFragment): unknown[][] {
  if (!Array.isArray(sample.rows)) {
    return [];
  }
  return sample.rows.filter(
    (row, index): row is unknown[] => Array.isArray(row) && (sample.levels[index] ?? 0) === 0,
  );
}

function sampleOf(sample: ReportPreviewFieldsFragment): ReportRunDeskSample {
  return {
    columns: sample.columns,
    rows: detailRowsOf(sample),
    totals: Array.isArray(sample.totals) ? sample.totals : null,
    truncated: sample.truncated,
  };
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
      ? sampleOf(getFragmentData(ReportPreviewFieldsFragmentDoc, card.sample))
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
