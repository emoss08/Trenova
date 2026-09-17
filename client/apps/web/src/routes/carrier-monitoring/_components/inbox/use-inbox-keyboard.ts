import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { useEffect, useRef } from "react";

export type InboxKeyboardHandlers = {
  onMove: (delta: 1 | -1) => void;
  onToggleSelect: () => void;
  onAcknowledge: () => void;
  onOpen: () => void;
  onClose: () => void;
};

export type InboxKeyAction = "next" | "previous" | "select" | "acknowledge" | "open" | "close";

export function inboxKeyAction(key: string): InboxKeyAction | null {
  switch (key) {
    case "j":
    case "ArrowDown":
      return "next";
    case "k":
    case "ArrowUp":
      return "previous";
    case "x":
      return "select";
    case "e":
      return "acknowledge";
    case "Enter":
      return "open";
    case "Escape":
      return "close";
    default:
      return null;
  }
}

function isInteractiveTarget(target: EventTarget | null): boolean {
  return (
    target instanceof Element &&
    target.closest('button, a[href], [role="checkbox"], [role="menuitem"], [role="radio"]') !== null
  );
}

/**
 * Triage moves faster on the keyboard: j/k walk the list, x selects, e acknowledges,
 * Enter opens the detail and Esc closes it. Keys typed into a field, a dialog or a
 * menu stay with that control.
 */
export function useInboxKeyboard(enabled: boolean, handlers: InboxKeyboardHandlers): void {
  const handlersRef = useRef(handlers);

  useEffect(() => {
    handlersRef.current = handlers;
  }, [handlers]);

  useEffect(() => {
    if (!enabled) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;

      const action = inboxKeyAction(event.key);
      if (!action) return;
      if ((action === "open" || action === "select") && isInteractiveTarget(event.target)) return;

      const current = handlersRef.current;
      switch (action) {
        case "next":
          event.preventDefault();
          current.onMove(1);
          break;
        case "previous":
          event.preventDefault();
          current.onMove(-1);
          break;
        case "select":
          event.preventDefault();
          current.onToggleSelect();
          break;
        case "acknowledge":
          event.preventDefault();
          current.onAcknowledge();
          break;
        case "open":
          event.preventDefault();
          current.onOpen();
          break;
        case "close":
          current.onClose();
          break;
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [enabled]);
}
