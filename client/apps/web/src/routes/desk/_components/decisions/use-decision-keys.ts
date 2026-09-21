import { useCallback, useEffect, useReducer } from "react";

export type QueueAsk = { kind: "accept" | "reject" | "modify"; ids: string[] };

export type QueueSelection = {
  focusedId: string | null;
  selectedIds: string[];
  /** A decision the keys asked for, waiting for the surface to carry it out. */
  pending: QueueAsk | null;
};

export function initialQueueSelection(): QueueSelection {
  return { focusedId: null, selectedIds: [], pending: null };
}

/** Rows that have left the queue are forgotten, wherever they were. */
function pruned(state: QueueSelection, ids: readonly string[]): QueueSelection {
  const known = new Set(ids);
  const selectedIds = state.selectedIds.filter((id) => known.has(id));
  const focusedId = state.focusedId !== null && known.has(state.focusedId) ? state.focusedId : null;
  if (selectedIds.length === state.selectedIds.length && focusedId === state.focusedId) {
    return state;
  }

  return { ...state, selectedIds, focusedId };
}

function step(state: QueueSelection, ids: readonly string[], direction: 1 | -1): QueueSelection {
  if (ids.length === 0) {
    return state;
  }
  const current = state.focusedId === null ? -1 : ids.indexOf(state.focusedId);
  const next =
    current === -1
      ? direction === 1
        ? 0
        : ids.length - 1
      : Math.min(ids.length - 1, Math.max(0, current + direction));

  return { ...state, focusedId: ids[next] };
}

function ask(state: QueueSelection, kind: QueueAsk["kind"], targets: string[]): QueueSelection {
  if (targets.length === 0) {
    return state;
  }

  return { ...state, pending: { kind, ids: targets } };
}

/**
 * The queue worked from the keyboard: j and k walk it, x marks the focused
 * row for a batch, a, r and m ask to accept, reject or modify the focused
 * row, Shift+A asks to accept every marked row (or the focused one when
 * nothing is marked), and Escape clears the marks and any ask. Rows that
 * have left the queue are dropped from the marks on every key. Pure, so the
 * whole session can be replayed in a test.
 */
export function reduceQueueKey(
  state: QueueSelection,
  key: string,
  ids: readonly string[],
): QueueSelection {
  const current = pruned(state, ids);
  const focused = current.focusedId !== null ? [current.focusedId] : [];

  switch (key) {
    case "j":
    case "ArrowDown":
      return step(current, ids, 1);
    case "k":
    case "ArrowUp":
      return step(current, ids, -1);
    case "x":
    case " ": {
      if (current.focusedId === null) {
        return current;
      }
      const marked = current.selectedIds.includes(current.focusedId);
      return {
        ...current,
        selectedIds: marked
          ? current.selectedIds.filter((id) => id !== current.focusedId)
          : [...current.selectedIds, current.focusedId],
      };
    }
    case "a":
      return ask(current, "accept", focused);
    case "r":
      return ask(current, "reject", focused);
    case "m":
      return ask(current, "modify", focused);
    case "A":
      return ask(current, "accept", current.selectedIds.length > 0 ? current.selectedIds : focused);
    case "R":
      return ask(current, "reject", current.selectedIds.length > 0 ? current.selectedIds : focused);
    case "Escape":
      return { ...current, selectedIds: [], pending: null };
    default:
      return state === current ? state : current;
  }
}

type QueueAction =
  | { type: "key"; key: string; ids: readonly string[] }
  | { type: "focus"; id: string | null }
  | { type: "toggle"; id: string; ids: readonly string[] }
  | { type: "select"; ids: string[] }
  | { type: "ask"; ask: QueueAsk | null }
  | { type: "prune"; ids: readonly string[] };

function queueReducer(state: QueueSelection, action: QueueAction): QueueSelection {
  switch (action.type) {
    case "key":
      return reduceQueueKey(state, action.key, action.ids);
    case "focus":
      return state.focusedId === action.id ? state : { ...state, focusedId: action.id };
    case "toggle": {
      const current = pruned(state, action.ids);
      const marked = current.selectedIds.includes(action.id);
      return {
        ...current,
        selectedIds: marked
          ? current.selectedIds.filter((id) => id !== action.id)
          : [...current.selectedIds, action.id],
      };
    }
    case "select":
      return { ...state, selectedIds: action.ids };
    case "ask":
      return { ...state, pending: action.ask };
    case "prune":
      return pruned(state, action.ids);
    default:
      return state;
  }
}

/** Whether a key press belongs to the page rather than to a field being typed in. */
function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  const tag = target.tagName;

  return (
    tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable === true
  );
}

/**
 * Binds the queue keys to the document while the queue is on screen, and
 * hands the surface a selection it can also drive with the pointer.
 */
export function useDecisionKeys(
  ids: readonly string[],
  options: { enabled?: boolean; onAsk: (ask: QueueAsk) => void },
) {
  const { enabled = true, onAsk } = options;
  const [selection, dispatch] = useReducer(queueReducer, undefined, initialQueueSelection);

  useEffect(() => {
    dispatch({ type: "prune", ids });
  }, [ids]);

  // A key that asks for a decision hands the ask to the surface from the
  // event itself; the reducer never keeps one waiting.
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey || isTypingTarget(event.target)) {
        return;
      }
      const before = selection;
      const after = reduceQueueKey(before, event.key, ids);
      if (after === before) {
        return;
      }
      event.preventDefault();
      if (after.pending !== null) {
        onAsk(after.pending);
      }
      dispatch({ type: "key", key: event.key, ids });
      if (after.pending !== null) {
        dispatch({ type: "ask", ask: null });
      }
    };
    document.addEventListener("keydown", onKeyDown);

    return () => document.removeEventListener("keydown", onKeyDown);
  }, [enabled, ids, onAsk, selection]);

  const focus = useCallback((id: string | null) => dispatch({ type: "focus", id }), []);
  const toggle = useCallback((id: string) => dispatch({ type: "toggle", id, ids }), [ids]);
  const select = useCallback((next: string[]) => dispatch({ type: "select", ids: next }), []);

  return { selection, focus, toggle, select };
}
