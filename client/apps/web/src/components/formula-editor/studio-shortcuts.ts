import { formatShortcut, isMacPlatform } from "@trenova/shared/lib/shortcuts";
import { useEffect } from "react";

export type StudioShortcut = "save" | "run" | "search" | "preview" | "scenarios" | "nova";

type KeyLike = {
  key: string;
  ctrlKey: boolean;
  metaKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  /** Whether the key went to something that takes text, where a bare key is typing. */
  typing?: boolean;
};

const KEY_TO_SHORTCUT: Record<string, StudioShortcut> = {
  s: "save",
  enter: "run",
  j: "nova",
  "1": "preview",
  "2": "scenarios",
};

const SHORTCUT_KEY_LABEL: Record<StudioShortcut, string> = {
  save: "S",
  run: "Enter",
  search: "/",
  preview: "1",
  scenarios: "2",
  nova: "J",
};

/**
 * Resolves a key press to a studio action. The platform's primary modifier
 * (Ctrl, or ⌘ on a Mac) plus a key counts; Shift or Alt on top means the
 * person wanted something else, such as a browser shortcut. A bare `/` opens
 * the reference search, unless it was typed into a field or the editor.
 */
export function matchStudioShortcut(event: KeyLike): StudioShortcut | null {
  const primary = event.ctrlKey || event.metaKey;
  if (event.shiftKey || event.altKey) return null;
  if (!primary) {
    return event.key === "/" && !event.typing ? "search" : null;
  }
  return KEY_TO_SHORTCUT[event.key.toLowerCase()] ?? null;
}

export function shortcutHint(shortcut: StudioShortcut, mac: boolean = isMacPlatform()): string {
  if (shortcut === "search") return SHORTCUT_KEY_LABEL.search;
  return formatShortcut(SHORTCUT_KEY_LABEL[shortcut], mac);
}

/** Whether an event's target takes text: a field, a text area, or an editable region such as the editor. */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

/**
 * Binds the studio shortcuts for the life of the component. Handlers that
 * are absent leave the browser's own behavior alone, so a page without a
 * search box does not swallow Ctrl+K.
 */
export function useStudioShortcuts(handlers: Partial<Record<StudioShortcut, () => void>>) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const shortcut = matchStudioShortcut({
        key: event.key,
        ctrlKey: event.ctrlKey,
        metaKey: event.metaKey,
        shiftKey: event.shiftKey,
        altKey: event.altKey,
        typing: isTypingTarget(event.target),
      });
      if (!shortcut) return;
      const handler = handlers[shortcut];
      if (!handler) return;
      event.preventDefault();
      handler();
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [handlers]);
}
