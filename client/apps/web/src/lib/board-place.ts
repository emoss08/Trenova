/** Where a board is drawn: in the conversation, as the reply itself, or beside it. */
export type BoardPlace = "conversation" | "beside";

/**
 * The tools whose call makes a board: show_board, the full page, and
 * show_answer, the answer in the conversation with the records it rests on,
 * which the server builds into a board.
 */
export const BOARD_CALL_TOOLS = ["show_board", "show_answer"] as const;

export type BoardCallTool = (typeof BOARD_CALL_TOOLS)[number];

export function isBoardCallTool(name: string): name is BoardCallTool {
  return (BOARD_CALL_TOOLS as readonly string[]).includes(name);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Where a stored board is drawn, read from its payload: the top-level `place`
 * a thread's artifact summary keeps, or `board.place` in the full payload. A
 * board stored before it had a place, or with any other value, is beside.
 */
export function boardPlaceOf(payload: Record<string, unknown>): BoardPlace {
  const board = isRecord(payload.board) ? payload.board : {};
  const place = payload.place ?? board.place;
  return place === "conversation" ? "conversation" : "beside";
}
