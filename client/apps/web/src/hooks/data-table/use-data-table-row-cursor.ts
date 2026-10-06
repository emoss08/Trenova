import { isTypingTarget, isWithinDialog, isWithinPopup } from "@/lib/dom";
import { useEffect, useRef } from "react";

export type DataTableRowCursorParams = {
  enabled: boolean;
  rowIds: readonly string[];
  cursorRowId: string | null;
  onCursorRowIdChange: (rowId: string | null) => void;
  expandedRowId: string | null;
  onExpandedRowIdChange?: (rowId: string | null) => void;
  hasSelection: boolean;
  onToggleSelect?: (rowId: string) => void;
  onClearSelection?: () => void;
};

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
      if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
      if (
        isTypingTarget(event.target) ||
        isWithinDialog(event.target) ||
        isWithinPopup(event.target)
      ) {
        return;
      }

      const {
        rowIds,
        cursorRowId,
        onCursorRowIdChange,
        expandedRowId,
        onExpandedRowIdChange,
        hasSelection,
        onToggleSelect,
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
        if (hasSelection && onClearSelection) {
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
