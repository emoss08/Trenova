import { keybindGroups } from "@/config/keybinds.config";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useEffect } from "react";

/**
 * Triage worked from the keyboard, the way a mail client is.
 *
 * The keys themselves live in `keybinds.config.ts` (group "triage"), so the
 * shortcuts sheet and the bindings are one list. A key resolves to a command
 * here, purely, from the key and a little of the page's state; the page
 * carries the command out through the one triage-actions hook. A key typed
 * into a field is text, and a key with a modifier belongs to the browser or
 * the app shell.
 */
export type TriageCommand =
  | "briefing"
  | "next"
  | "previous"
  | "search"
  | "togglePanel"
  | "doNextStep"
  | "handle"
  | "ignore"
  | "reply"
  | "snooze"
  | "assignToMe"
  | "toggleSelect"
  | "link"
  | "askDesk"
  | "shortcuts"
  | "clearSelection"
  | "closePanel"
  | "closeMessage";

const KEYBIND_COMMAND: Readonly<Record<string, TriageCommand | "escape">> = {
  "triage-briefing": "briefing",
  "triage-next": "next",
  "triage-previous": "previous",
  "triage-search": "search",
  "triage-panel": "togglePanel",
  "triage-do": "doNextStep",
  "triage-handle": "handle",
  "triage-ignore": "ignore",
  "triage-reply": "reply",
  "triage-snooze": "snooze",
  "triage-assign": "assignToMe",
  "triage-select": "toggleSelect",
  "triage-link": "link",
  "triage-desk": "askDesk",
  "triage-shortcuts": "shortcuts",
  "triage-escape": "escape",
};

/** The arrows do what J and K do; they are not separate entries in the sheet. */
const ALIASES: Readonly<Record<string, TriageCommand>> = {
  ArrowDown: "next",
  ArrowUp: "previous",
};

function eventKeyOf(configKey: string): string {
  if (configKey === "Esc") {
    return "Escape";
  }

  return configKey.length === 1 ? configKey.toLowerCase() : configKey;
}

/** event.key → what it does, read from the keybinds config. */
export const TRIAGE_KEYMAP: ReadonlyMap<string, TriageCommand | "escape"> = new Map([
  ...(keybindGroups.find((group) => group.id === "triage")?.keybinds ?? []).flatMap(
    (keybind): Array<[string, TriageCommand | "escape"]> => {
      const command = KEYBIND_COMMAND[keybind.id];
      return command === undefined || keybind.keys.length !== 1
        ? []
        : [[eventKeyOf(keybind.keys[0]), command]];
    },
  ),
  ...Object.entries(ALIASES),
]);

/** Commands that act on the open message, or on the selection when there is one. */
const ON_SELECTION: ReadonlySet<TriageCommand> = new Set([
  "handle",
  "ignore",
  "snooze",
  "assignToMe",
]);

/** Commands that need a message open. */
const ON_OPEN: ReadonlySet<TriageCommand> = new Set([
  "togglePanel",
  "doNextStep",
  "reply",
  "toggleSelect",
  "link",
  "askDesk",
]);

/** Commands that only mean something in a folder of the queue. */
const IN_QUEUE: ReadonlySet<TriageCommand> = new Set(["next", "previous", "search"]);

export type KeyPress = {
  key: string;
  metaKey: boolean;
  ctrlKey: boolean;
  altKey: boolean;
  /** The key landed in a field, an editable region or a select. */
  editable: boolean;
};

export type KeyContext = {
  /** A folder of the queue is showing (not the Briefing or Rules). */
  inQueue: boolean;
  hasOpenMessage: boolean;
  selectionSize: number;
  panelOpen: boolean;
  /** A dialog, sheet or menu is up and owns the keyboard. */
  overlayOpen: boolean;
};

/** What a key press does, or null when Triage leaves it alone. */
export function resolveTriageKey(press: KeyPress, context: KeyContext): TriageCommand | null {
  if (press.metaKey || press.ctrlKey || press.altKey || press.editable || context.overlayOpen) {
    return null;
  }
  const mapped = TRIAGE_KEYMAP.get(press.key.length === 1 ? press.key.toLowerCase() : press.key);
  if (mapped === undefined) {
    return null;
  }

  if (mapped === "escape") {
    if (context.selectionSize > 0) {
      return "clearSelection";
    }
    if (context.hasOpenMessage && context.panelOpen) {
      return "closePanel";
    }
    return context.hasOpenMessage ? "closeMessage" : null;
  }
  if (IN_QUEUE.has(mapped) && !context.inQueue) {
    return null;
  }
  if (ON_SELECTION.has(mapped)) {
    return context.hasOpenMessage || context.selectionSize > 0 ? mapped : null;
  }
  if (ON_OPEN.has(mapped)) {
    return context.hasOpenMessage ? mapped : null;
  }

  return mapped;
}

/** Where the open message moves on J or K: the first at either end when nothing is open. */
export function stepFrom(
  ids: readonly string[],
  openId: string | null,
  direction: 1 | -1,
): string | null {
  if (ids.length === 0) {
    return null;
  }
  const current = openId === null ? -1 : ids.indexOf(openId);
  if (current === -1) {
    return direction === 1 ? ids[0] : ids[ids.length - 1];
  }

  return ids[Math.min(ids.length - 1, Math.max(0, current + direction))];
}

/**
 * Where the reader goes when messages leave the folder: the first message
 * after the open one that is staying, else the nearest before it, else
 * nothing. An open message that is not leaving stays open.
 */
export function nextAfterLeaving(
  ids: readonly string[],
  leaving: ReadonlySet<string>,
  openId: string | null,
): string | null {
  if (openId === null || !leaving.has(openId)) {
    return openId;
  }
  const index = ids.indexOf(openId);
  if (index === -1) {
    return null;
  }
  for (let next = index + 1; next < ids.length; next += 1) {
    if (!leaving.has(ids[next])) {
      return ids[next];
    }
  }
  for (let previous = index - 1; previous >= 0; previous -= 1) {
    if (!leaving.has(ids[previous])) {
      return ids[previous];
    }
  }

  return null;
}

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
 * Listens on the document while Triage is mounted and hands each command to
 * `dispatch`. `context` is read when a key is pressed, so it may change on
 * every render without re-binding the listener. Escape in a field leaves the
 * field, the way it does in every mail client.
 */
export function useTriageKeys({
  enabled,
  context,
  dispatch,
}: {
  enabled: boolean;
  context: () => KeyContext;
  dispatch: (command: TriageCommand) => void;
}): void {
  const readContext = useLatestCallback(context);
  const run = useLatestCallback(dispatch);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented) {
        return;
      }
      const editable = isEditable(event.target);
      if (editable && event.key === "Escape" && event.target instanceof HTMLElement) {
        event.target.blur();
        return;
      }
      const command = resolveTriageKey(
        {
          key: event.key,
          metaKey: event.metaKey,
          ctrlKey: event.ctrlKey,
          altKey: event.altKey,
          editable,
        },
        readContext(),
      );
      if (command === null) {
        return;
      }
      if (command === "doNextStep" && event.target instanceof HTMLButtonElement) {
        return;
      }
      event.preventDefault();
      run(command);
    }

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [enabled, readContext, run]);
}
