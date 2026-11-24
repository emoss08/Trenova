import type { DeskBoardCardQuery } from "@trenova/graphql/generated/graphql";
import { DeskBoardCardDocument } from "@trenova/graphql/generated/documents/agent/desk-board";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { analysisTableSchema, type AnalysisTable } from "@/types/analysis";

type RawCard = DeskBoardCardQuery["deskBoardCard"];
type RawBlock = RawCard["blocks"][number];

export type DeskBoardValue = NonNullable<RawBlock["figures"][number]["value"]>;
export type DeskBoardSegment = RawCard["verdict"][number];
export type DeskBoardRow = RawBlock["rows"][number];
export type DeskBoardRowMeasure = NonNullable<DeskBoardRow["measure"]>;
/** Words the server wrote in the reader's language, with `{0}` where a value or a moment goes. */
export type DeskBoardWords = DeskBoardRowMeasure["standing"];
/** A row's bar, laid out by the server on one scale for its whole block. */
export type DeskBoardBar = NonNullable<DeskBoardRowMeasure["bar"]>;
/** A chart or metric block's read, drawn in its form. */
export type DeskBoardMetric = NonNullable<RawBlock["metric"]>;
export type DeskBoardFigure = RawBlock["figures"][number];
export type DeskBoardCompare = NonNullable<RawBlock["compare"]>;
export type DeskBoardEvent = RawBlock["events"][number];
export type DeskBoardCheck = RawBlock["checks"][number];
export type DeskBoardMeter = NonNullable<RawBlock["meter"]>;
export type DeskBoardSource = RawBlock["source"];
/** A figure the board's words place: its value now, what it was when answered, and its source. */
export type DeskBoardReading = RawCard["readings"][number];

/** A board's table block: the first rows of the table, and how many it holds. */
export type DeskBoardTable = {
  table: AnalysisTable;
  total: number;
  artifactId: string | null;
};

/** One block of a board, its table read into the analysis shape it is kept in. */
export type DeskBoardBlock = Omit<RawBlock, "table"> & {
  table: DeskBoardTable | null;
};

/** A board as the Desk draws it, read now with the reader's access. */
export type DeskBoardCard = Omit<RawCard, "blocks"> & { blocks: DeskBoardBlock[] };

function tableOf(raw: RawBlock["table"]): DeskBoardTable | null {
  if (!raw) return null;
  const parsed = analysisTableSchema.safeParse(raw.table);
  if (!parsed.success) return null;
  return { table: parsed.data, total: raw.total, artifactId: raw.artifactId ?? null };
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
    })),
  };
}
