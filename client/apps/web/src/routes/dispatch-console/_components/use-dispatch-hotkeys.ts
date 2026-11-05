import { isTypingTarget, isWithinDialog, isWithinPopup, mayMoveFocusWithin } from "@/lib/dom";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { useEffect } from "react";
import { queueRowId } from "./queue-ids";

type DispatchHotkeyParams = {
  /** False while a dialog owns the screen, so the board never moves underneath it. */
  enabled: boolean;
  /** The moves the queue shows, in the order it draws them, which is the order J and K walk. */
  orderedMoves: readonly DispatchBoardMove[];
  selectedMoveId: string | null;
  canUndo: boolean;
  /** The queue's own list: its keys are the console's even though it is a listbox. */
  queueScope: Element | null;
  onSelectMove: (moveId: string) => void;
  onClearSelection: () => void;
  onShiftWindow: (direction: 1 | -1) => void;
  onToday: () => void;
  onUndo: () => void;
};

/** The move J or K lands on: the next or previous one shown, or the first when none is open. */
export function stepMove(
  orderedMoves: readonly DispatchBoardMove[],
  selectedMoveId: string | null,
  delta: 1 | -1,
): string | null {
  if (orderedMoves.length === 0) return null;
  const index = orderedMoves.findIndex((move) => move.moveId === selectedMoveId);
  if (index === -1) return orderedMoves[0].moveId;
  const next = Math.min(Math.max(index + delta, 0), orderedMoves.length - 1);
  return orderedMoves[next].moveId;
}

/** Moves focus to a queue row once it is mounted, unless the person is busy elsewhere. */
export function focusQueueRow(moveId: string, scope: Element | null): void {
  if (!mayMoveFocusWithin(scope)) return;
  // The row may mount only after the list scrolls to it, a frame later.
  requestAnimationFrame(() =>
    requestAnimationFrame(() =>
      document.getElementById(queueRowId(moveId))?.focus({ preventScroll: true }),
    ),
  );
}

/**
 * Dispatchers cover a board faster on the keyboard than with a mouse. The inspector owns
 * A, D and the digits; these are the page's: J and K walk the queue, U undoes, T, ← and →
 * move the window, and Esc closes what is open.
 */
export function useDispatchHotkeys({
  enabled,
  orderedMoves,
  selectedMoveId,
  canUndo,
  queueScope,
  onSelectMove,
  onClearSelection,
  onShiftWindow,
  onToday,
  onUndo,
}: DispatchHotkeyParams): void {
  useEffect(() => {
    if (!enabled) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;
      if (
        isWithinPopup(event.target) &&
        !(queueScope && event.target instanceof Node && queueScope.contains(event.target))
      ) {
        return;
      }

      const key = event.key;
      if ((key === "u" || key === "U") && canUndo) {
        event.preventDefault();
        onUndo();
        return;
      }
      if (key === "t" || key === "T") {
        onToday();
        return;
      }
      if (key === "ArrowLeft") {
        event.preventDefault();
        onShiftWindow(-1);
        return;
      }
      if (key === "ArrowRight") {
        event.preventDefault();
        onShiftWindow(1);
        return;
      }
      if (key === "Escape") {
        onClearSelection();
        return;
      }
      if (key === "j" || key === "k" || key === "J" || key === "K") {
        const next = stepMove(orderedMoves, selectedMoveId, key.toLowerCase() === "j" ? 1 : -1);
        if (!next) return;
        event.preventDefault();
        onSelectMove(next);
        focusQueueRow(next, queueScope);
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [
    canUndo,
    enabled,
    onClearSelection,
    onSelectMove,
    onShiftWindow,
    onToday,
    onUndo,
    orderedMoves,
    queueScope,
    selectedMoveId,
  ]);
}
