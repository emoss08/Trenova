import { isTypingTarget, isWithinDialog, isWithinPopup } from "@/lib/dom";
import { useEffect, useRef } from "react";

export type DataTableRowCursorShortcut = {
  /** The lowercase key; an Alt chord is matched by physical key so Option on a Mac still works. */
  key: string;
  mod?: boolean;
  alt?: boolean;
  run: (rowId: string) => void;
};

export type DataTableRowCursorParams = {
  enabled: boolean;
  rowIds: readonly string[];
  cursorRowId: string | null;
  onCursorRowIdChange: (rowId: string | null) => void;
  expandedRowId: string | null;
  onExpandedRowIdChange?: (rowId: string | null) => void;
  /** Asked when Escape is pressed, so the selection need not be followed between keys. */
  hasSelection: () => boolean;
  onToggleSelect?: (rowId: string) => void;
  /** Pins the row under the cursor to the top of the table, or unpins it. */
  onTogglePin?: (rowId: string) => void;
  onClearSelection?: () => void;
  shortcuts?: readonly DataTableRowCursorShortcut[];
};

function matchesShortcut(event: KeyboardEvent, shortcut: DataTableRowCursorShortcut) {
  const mod = event.metaKey || event.ctrlKey;
  if (!!shortcut.mod !== mod || !!shortcut.alt !== event.altKey || event.shiftKey) return false;
  if (shortcut.alt) {
    return event.code === `Key${shortcut.key.toUpperCase()}`;
  }
  return event.key.toLowerCase() === shortcut.key;
}

const DOWN_KEYS = new Set(["j", "ArrowDown"]);
const UP_KEYS = new Set(["k", "ArrowUp"]);

/**
 * Page-level row cursor for a table. The keys are read off the window so the
 * cursor works wherever focus rests on the page, and they stand aside for a
 * field, a dialog, an open menu or any modified chord, which keeps the
 * application's own shortcuts and typing intact.
 */
export function useDataTableRowCursor(params: DataTableRowCursorParams) {
  const latest = useRef(params);
  latest.current = params;

  useEffect(() => {
    if (!params.enabled) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented) return;
      if (
        isTypingTarget(event.target) ||
        isWithinDialog(event.target) ||
        isWithinPopup(event.target)
      ) {
        return;
      }

      const shortcut = latest.current.shortcuts?.find((entry) => matchesShortcut(event, entry));
      if (shortcut) {
        const target = latest.current.cursorRowId ?? latest.current.expandedRowId;
        if (!target) return;
        event.preventDefault();
        shortcut.run(target);
        return;
      }
      if (event.metaKey || event.ctrlKey || event.altKey) return;

      const {
        rowIds,
        cursorRowId,
        onCursorRowIdChange,
        expandedRowId,
        onExpandedRowIdChange,
        hasSelection,
        onToggleSelect,
        onTogglePin,
        onClearSelection,
      } = latest.current;

      const anchor = cursorRowId ?? expandedRowId;
      const anchorIndex = anchor ? rowIds.indexOf(anchor) : -1;

      if (DOWN_KEYS.has(event.key) || UP_KEYS.has(event.key)) {
        if (rowIds.length === 0) return;
        event.preventDefault();
        const step = DOWN_KEYS.has(event.key) ? 1 : -1;
        const nextIndex =
          anchorIndex < 0 ? 0 : Math.max(0, Math.min(rowIds.length - 1, anchorIndex + step));
        const nextId = rowIds[nextIndex];
        onCursorRowIdChange(nextId);
        if (expandedRowId && expandedRowId !== nextId) {
          onExpandedRowIdChange?.(nextId);
        }
        return;
      }

      if (event.key === "Enter") {
        if (anchorIndex < 0 || !onExpandedRowIdChange) return;
        event.preventDefault();
        const id = rowIds[anchorIndex];
        onExpandedRowIdChange(expandedRowId === id ? null : id);
        if (cursorRowId !== id) onCursorRowIdChange(id);
        return;
      }

      if (event.key === "p" || event.key === "P") {
        if (anchorIndex < 0 || !onTogglePin) return;
        event.preventDefault();
        onTogglePin(rowIds[anchorIndex]);
        return;
      }

      if (event.key === "x" || event.key === "X") {
        if (anchorIndex < 0 || !onToggleSelect) return;
        event.preventDefault();
        onToggleSelect(rowIds[anchorIndex]);
        return;
      }

      if (event.key === "Escape") {
        if (expandedRowId && onExpandedRowIdChange) {
          event.preventDefault();
          onExpandedRowIdChange(null);
          return;
        }
        if (onClearSelection && hasSelection()) {
          event.preventDefault();
          onClearSelection();
          return;
        }
        if (cursorRowId) {
          event.preventDefault();
          onCursorRowIdChange(null);
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [params.enabled]);
}
