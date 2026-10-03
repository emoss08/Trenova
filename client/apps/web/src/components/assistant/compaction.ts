import type { AssistantCompactionEvent, ContextUsage } from "@/types/assistant";

/** How full a conversation gets before it compacts itself; the server holds the same share. */
export const AUTO_COMPACT_SHARE = 0.85;

/** The least a compaction has to free to be offered. */
export const MIN_COMPACTION_TOKENS = 4000;

/** The most a compaction summary may be, which the server also holds. */
export const SUMMARY_TOKEN_BUDGET = 1200;

/** The window shown before a conversation has been measured. */
export const DEFAULT_CONTEXT_WINDOW = 200_000;

/** How long the meter in the status line takes to drain to its new figure. */
export const DRAIN_MS = 2600;

export type MeterTone = "" | "warm" | "hot";

/** One part of the context, as the meter's bar and legend draw it. */
export type MeterPart = {
  key: "conv" | "tools" | "files" | "sys";
  tokens: number;
  /** The bar's colour class. */
  tone: "c1" | "c2" | "c3" | "c4";
};

/** What the composer's context meter shows. */
export type MeterView = {
  used: number;
  window: number;
  /** How full, from 0; past 1 for a conversation that outgrew its model. */
  share: number;
  pct: number;
  tone: MeterTone;
  /** The figure beside the ring, shown from three quarters full. */
  showPct: boolean;
  parts: MeterPart[];
  /** Roughly what compacting now would give back. */
  frees: number;
  canCompact: boolean;
};

/** A summary of a stretch is expected to be a fifth of it, never more than the budget. */
export function summaryEstimate(tokens: number): number {
  return Math.min(Math.floor(tokens / 5), SUMMARY_TOKEN_BUDGET);
}

export function usageTotal(usage: ContextUsage | null | undefined): number {
  if (!usage) {
    return 0;
  }
  return usage.instructions + usage.messages + usage.toolResults + usage.files;
}

/**
 * What the meter shows for a conversation's last measure. A conversation not
 * yet measured shows an empty ring against the default window, as a new
 * conversation is.
 */
export function meterView(usage: ContextUsage | null | undefined): MeterView {
  const window = usage && usage.window > 0 ? usage.window : DEFAULT_CONTEXT_WINDOW;
  const used = usageTotal(usage);
  const share = used / window;
  const compactable = usage?.compactable ?? 0;
  const frees = compactable - summaryEstimate(compactable);

  return {
    used,
    window,
    share,
    pct: Math.round(share * 100),
    tone: share >= 0.9 ? "hot" : share >= 0.75 ? "warm" : "",
    showPct: share >= 0.75,
    parts: [
      { key: "conv", tokens: usage?.messages ?? 0, tone: "c1" },
      { key: "tools", tokens: usage?.toolResults ?? 0, tone: "c2" },
      { key: "files", tokens: usage?.files ?? 0, tone: "c3" },
      { key: "sys", tokens: usage?.instructions ?? 0, tone: "c4" },
    ],
    frees,
    canCompact: frees > MIN_COMPACTION_TOKENS,
  };
}

/**
 * A token count as the meter writes it: 41.8k, 200k, 950. Thousands keep
 * one decimal below a hundred thousand.
 */
export function kfmt(tokens: number): string {
  if (tokens < 1000) {
    return String(tokens);
  }
  const fixed = (tokens / 1000).toFixed(tokens >= 100_000 ? 0 : 1).replace(/\.0$/, "");
  return `${fixed}k`;
}

/**
 * Where a conversation's compaction stands, as the composer shows it.
 *
 * - idle: nothing is compacting. `ignored` is a compaction the person
 *   cancelled, whose late events change nothing; `error` says why the last
 *   one did not finish.
 * - compacting: a compaction is under way. Its turn is null between the
 *   person asking and the server naming the turn.
 */
export type CompactionState =
  | { phase: "idle"; ignored: string | null; error: string | null }
  | {
      phase: "compacting";
      turnId: string | null;
      auto: boolean;
      before: number;
      after: number;
    };

export type CompactionAction =
  /** The person asked; the figures are the meter's own until the server's arrive. */
  | { type: "request"; before: number; after: number }
  /** The server named the turn compacting, from its own stream or the turn that set it off. */
  | { type: "started"; event: AssistantCompactionEvent }
  /** Reopening a conversation found a compaction under way. */
  | { type: "rejoined"; turnId: string; before: number; after: number }
  | { type: "finished"; turnId: string }
  | { type: "cancelled"; turnId: string }
  /** The person pressed Cancel: the composer frees at once, and the turn's late events are ignored. */
  | { type: "cancel" }
  | { type: "failed"; message: string };

export const IDLE: CompactionState = { phase: "idle", ignored: null, error: null };

/**
 * Folds what happens to a compaction into what the composer shows. Events for
 * a turn the person already cancelled, or for another turn than the one being
 * shown, change nothing.
 */
export function compactionReducer(
  state: CompactionState,
  action: CompactionAction,
): CompactionState {
  switch (action.type) {
    case "request":
      if (state.phase === "compacting") {
        return state;
      }
      return {
        phase: "compacting",
        turnId: null,
        auto: false,
        before: action.before,
        after: action.after,
      };

    case "started": {
      const { event } = action;
      if (state.phase === "idle" && state.ignored === event.turnId) {
        return state;
      }
      if (state.phase === "compacting" && state.turnId !== null && state.turnId !== event.turnId) {
        return state;
      }
      // The server's figures replace the meter's own estimate; a stream that
      // names the turn without them keeps what is shown.
      const shown = state.phase === "compacting" ? state : null;
      return {
        phase: "compacting",
        turnId: event.turnId,
        auto: event.auto,
        before: event.before > 0 ? event.before : (shown?.before ?? 0),
        after: event.after > 0 ? event.after : (shown?.after ?? 0),
      };
    }

    case "rejoined":
      if (state.phase === "compacting" || state.ignored === action.turnId) {
        return state;
      }
      return {
        phase: "compacting",
        turnId: action.turnId,
        auto: false,
        before: action.before,
        after: action.after,
      };

    case "finished":
    case "cancelled":
      if (state.phase !== "compacting") {
        return state;
      }
      if (state.turnId !== null && state.turnId !== action.turnId) {
        return state;
      }
      return IDLE;

    case "cancel":
      if (state.phase !== "compacting") {
        return state;
      }
      return { phase: "idle", ignored: state.turnId, error: null };

    case "failed":
      if (state.phase !== "compacting") {
        return state;
      }
      return { phase: "idle", ignored: null, error: action.message };
  }
}
