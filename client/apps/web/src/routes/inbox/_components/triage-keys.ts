import { useEffect } from "react";

/**
 * The inbox worked from the keyboard, the way a mail client is.
 *
 * j and k (or the arrows) open the next and previous message — in a reading
 * pane, moving is opening. e marks the open message handled, # ignores it,
 * l links it by hand, a asks the desk about it, and Escape closes the pane.
 * A key names what to do; the page carries it out, so the whole session is a
 * pure fold a test can replay.
 */
export type TriageAskKind = "handle" | "ignore" | "link" | "ask";

export type TriageState = {
  openId: string | null;
  ask: { kind: TriageAskKind; id: string } | null;
};

const ASK_KEYS: Record<string, TriageAskKind> = {
  e: "handle",
  "#": "ignore",
  l: "link",
  a: "ask",
};

function step(ids: readonly string[], openId: string | null, direction: 1 | -1): string | null {
  if (ids.length === 0) {
    return null;
  }
  const current = openId === null ? -1 : ids.indexOf(openId);
  if (current === -1) {
    return direction === 1 ? ids[0] : ids[ids.length - 1];
  }

  return ids[Math.min(ids.length - 1, Math.max(0, current + direction))];
}

export function reduceTriageKey(
  state: TriageState,
  key: string,
  ids: readonly string[],
): TriageState {
  const openId = state.openId !== null && ids.includes(state.openId) ? state.openId : null;

  switch (key) {
    case "j":
    case "ArrowDown":
      return { openId: step(ids, openId, 1), ask: null };
    case "k":
    case "ArrowUp":
      return { openId: step(ids, openId, -1), ask: null };
    case "Escape":
      return { openId: null, ask: null };
  }

  const askKind = ASK_KEYS[key];
  if (askKind !== undefined) {
    return { openId, ask: openId === null ? null : { kind: askKind, id: openId } };
  }

  return openId === state.openId ? state : { ...state, openId };
}

/** Where the reader goes when the open message leaves the list. */
export function nextAfterLeaving(ids: readonly string[], leftId: string): string | null {
  const index = ids.indexOf(leftId);
  if (index === -1) {
    return null;
  }

  return ids[index + 1] ?? ids[index - 1] ?? null;
}

/** Typing into a field is typing, not a command. */
function isEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}

/**
 * Listens on the document while the inbox is mounted. A key with a modifier
 * belongs to the browser or the app shell, and a key typed into a field is
 * text; neither is a command. "/" is the one key that reaches from anywhere
 * on the page into the search box, as it does in every mail client.
 */
export function useTriageKeys({
  enabled,
  onKey,
  onSearch,
}: {
  enabled: boolean;
  onKey: (key: string) => void;
  onSearch: () => void;
}) {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey || event.defaultPrevented) {
        return;
      }
      if (isEditable(event.target)) {
        if (event.key === "Escape" && event.target instanceof HTMLElement) {
          event.target.blur();
        }
        return;
      }
      if (event.key === "/") {
        event.preventDefault();
        onSearch();
        return;
      }
      if (["j", "k", "ArrowDown", "ArrowUp", "Escape", "e", "#", "l", "a"].includes(event.key)) {
        event.preventDefault();
        onKey(event.key);
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onKey, onSearch]);
}
