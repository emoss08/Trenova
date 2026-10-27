import { DeskBoardCardDocument, type DeskBoardCardQuery } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import {
  analysisChartSchema,
  analysisTableSchema,
  type AnalysisChart,
  type AnalysisTable,
} from "@/types/analysis";

type RawCard = DeskBoardCardQuery["deskBoardCard"];
type RawBlock = RawCard["blocks"][number];

export type DeskBoardValue = NonNullable<RawBlock["figures"][number]["value"]>;
export type DeskBoardSegment = RawCard["verdict"][number];
export type DeskBoardRow = RawBlock["rows"][number];
export type DeskBoardRowMeasure = NonNullable<DeskBoardRow["measure"]>;
export type DeskBoardFigure = RawBlock["figures"][number];
export type DeskBoardCompare = NonNullable<RawBlock["compare"]>;
export type DeskBoardEvent = RawBlock["events"][number];
export type DeskBoardCheck = RawBlock["checks"][number];
export type DeskBoardMeter = NonNullable<RawBlock["meter"]>;
export type DeskBoardSource = RawBlock["source"];

/** A board's table block: the first rows of the table, and how many it holds. */
export type DeskBoardTable = {
  table: AnalysisTable;
  total: number;
  artifactId: string | null;
};

/** One block of a board, its table and chart read into the analysis shapes they are kept in. */
export type DeskBoardBlock = Omit<RawBlock, "table" | "chart"> & {
  table: DeskBoardTable | null;
  chart: AnalysisChart | null;
};

/** A board as the Desk draws it, read now with the reader's access. */
export type DeskBoardCard = Omit<RawCard, "blocks"> & { blocks: DeskBoardBlock[] };

function tableOf(raw: RawBlock["table"]): DeskBoardTable | null {
  if (!raw) return null;
  const parsed = analysisTableSchema.safeParse(raw.table);
  if (!parsed.success) return null;
  return { table: parsed.data, total: raw.total, artifactId: raw.artifactId ?? null };
}

function chartOf(raw: RawBlock["chart"]): AnalysisChart | null {
  if (!raw) return null;
  const parsed = analysisChartSchema.safeParse(raw);
  return parsed.success ? parsed.data : null;
}

/** A board in one of the person's conversations, read now: one query fills every block. */
export async function fetchDeskBoardCard(
  threadId: string,
  artifactId: string,
  options?: { signal?: AbortSignal },
): Promise<DeskBoardCard> {
  const data = await requestGraphQL({
    document: DeskBoardCardDocument,
    operationName: "DeskBoardCard",
    variables: { threadId, artifactId },
    signal: options?.signal,
  });
  const card = data.deskBoardCard;
  return {
    ...card,
    blocks: card.blocks.map((block) => ({
      ...block,
      table: tableOf(block.table),
      chart: chartOf(block.chart),
    })),
  };
}
