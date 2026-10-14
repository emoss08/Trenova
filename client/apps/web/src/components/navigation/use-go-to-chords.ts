import { useEffect } from "react";
import { useNavigate } from "react-router";
import { GO_CHORD_KEY, GO_CHORD_WINDOW_MS } from "@/components/navigation/rail-shortcuts";

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return (
    target.closest("input, textarea, select, [contenteditable=''], [contenteditable='true']") !==
    null
  );
}

/**
 * G, then a module's letter, goes there. The chord is armed for a moment and
 * ignored while a field has focus, so typing a g into a search box never
 * moves the page.
 */
export function useGoToChords(targets: ReadonlyMap<string, string>) {
  const navigate = useNavigate();

  useEffect(() => {
    let armedUntil = 0;

    function onKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey || isTypingTarget(event.target)) {
        return;
      }
      const key = event.key.length === 1 ? event.key.toUpperCase() : event.key;
      const now = performance.now();

      if (armedUntil > now) {
        armedUntil = 0;
        const path = targets.get(key);
        if (path) {
          event.preventDefault();
          void navigate(path);
        }
        return;
      }

      if (key === GO_CHORD_KEY && !event.shiftKey) {
        armedUntil = now + GO_CHORD_WINDOW_MS;
      }
    }

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [targets, navigate]);
}
