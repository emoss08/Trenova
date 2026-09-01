import { useEffect } from "react";

export type StudioShortcut = "save" | "run" | "search" | "preview" | "scenarios";

type KeyLike = {
  key: string;
  ctrlKey: boolean;
  metaKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
};

const KEY_TO_SHORTCUT: Record<string, StudioShortcut> = {
  s: "save",
  enter: "run",
  k: "search",
  "1": "preview",
  "2": "scenarios",
};

const SHORTCUT_KEY_LABEL: Record<StudioShortcut, string> = {
  save: "S",
  run: "Enter",
  search: "K",
  preview: "1",
  scenarios: "2",
};

/**
 * Resolves a key press to a studio action. Only the platform's primary
 * modifier (Ctrl, or ⌘ on a Mac) plus the key counts; Shift or Alt on top
 * means the person wanted something else, such as a browser shortcut.
 */
export function matchStudioShortcut(event: KeyLike): StudioShortcut | null {
  const primary = event.ctrlKey || event.metaKey;
  if (!primary || event.shiftKey || event.altKey) return null;
  return KEY_TO_SHORTCUT[event.key.toLowerCase()] ?? null;
}

export function isMacPlatform(): boolean {
  if (typeof navigator === "undefined") return false;
  return /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
}

export function shortcutHint(shortcut: StudioShortcut, mac: boolean = isMacPlatform()): string {
  const key = SHORTCUT_KEY_LABEL[shortcut];
  return mac ? `⌘${key}` : `Ctrl+${key}`;
}

/**
 * Binds the studio shortcuts for the life of the component. Handlers that
 * are absent leave the browser's own behavior alone, so a page without a
 * search box does not swallow Ctrl+K.
 */
export function useStudioShortcuts(handlers: Partial<Record<StudioShortcut, () => void>>) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const shortcut = matchStudioShortcut(event);
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
