import type { ApprovalTarget } from "@/lib/graphql/agent-decisions";

/** The undo window, in whole seconds, as the ring counts it. */
export const UNDO_SECONDS = 5;

/**
 * An approval in its undo window: what was approved, what the server is
 * about to do, and when it will. The server holds the approval and commits
 * it at `commitsAt`; this only draws the countdown towards it.
 */
export type UndoWindow = {
  key: string;
  /** What was approved, as the card titled it. */
  title: string;
  /** What will happen when the window closes, e.g. "Assign biller on 11 items". */
  what: string;
  target: ApprovalTarget;
  /** The proposals the approval holds, to put back on undo. */
  proposalIds: readonly string[];
  planId: string | null;
  /** When the server commits, in Unix seconds; null until it has answered. */
  commitsAt: number | null;
  /** When the person approved, by this browser's clock. */
  startedAt: number;
};

export type UndoState =
  | { phase: "idle" }
  | { phase: "waiting"; window: UndoWindow }
  /** Gone through: the green row holds a moment before the composer is free. */
  | { phase: "committed"; window: UndoWindow };

export type UndoAction =
  | { type: "start"; window: UndoWindow }
  /** The server took the approval; null when it went through at once. */
  | { type: "scheduled"; key: string; commitsAt: number | null }
  /** The server refused the approval, or it was undone: the card comes back. */
  | { type: "clear"; key: string }
  | { type: "commit"; key: string }
  | { type: "tick"; now: number };

export const IDLE: UndoState = { phase: "idle" };

/**
 * Seconds left on the ring. The deadline is the server's once it has
 * answered, so the ring follows the commit rather than the click; before
 * that it counts from the click. A clock far off the server's is held to the
 * window, so the ring never shows more than it or less than nothing.
 */
export function secondsLeft(window: UndoWindow, now: number): number {
  const deadline =
    window.commitsAt !== null ? window.commitsAt * 1000 : window.startedAt + UNDO_SECONDS * 1000;
  const left = Math.ceil((deadline - now) / 1000);

  return Math.max(0, Math.min(UNDO_SECONDS, left));
}

export function undoReducer(state: UndoState, action: UndoAction): UndoState {
  switch (action.type) {
    case "start":
      return { phase: "waiting", window: action.window };
    case "scheduled":
      if (state.phase !== "waiting" || state.window.key !== action.key) {
        return state;
      }
      // An approval the server carried out at once has no window to wait.
      if (action.commitsAt === null) {
        return { phase: "committed", window: state.window };
      }
      return { phase: "waiting", window: { ...state.window, commitsAt: action.commitsAt } };
    case "clear":
      return state.phase !== "idle" && state.window.key === action.key ? IDLE : state;
    case "commit":
      return state.phase === "waiting" && state.window.key === action.key
        ? { phase: "committed", window: state.window }
        : state;
    case "tick":
      // The ring reaching nothing before the server answered is not a commit:
      // the approval may yet be refused.
      if (
        state.phase === "waiting" &&
        state.window.commitsAt !== null &&
        secondsLeft(state.window, action.now) === 0
      ) {
        return { phase: "committed", window: state.window };
      }
      return state;
  }
}
