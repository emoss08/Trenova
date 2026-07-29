import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { useEffect } from "react";

type DispatchHotkeyParams = {
  /** False while a dialog owns the screen, so the board never moves underneath it. */
  enabled: boolean;
  /** The moves in the order the board draws them, which is the order j/k walks. */
  orderedMoves: readonly DispatchBoardMove[];
  selectedMoveId: string | null;
  canUndo: boolean;
  onSelectMove: (moveId: string) => void;
  onClearSelection: () => void;
  onShiftWindow: (direction: 1 | -1) => void;
  onToday: () => void;
  onUndo: () => void;
};

/**
 * Dispatchers cover a board faster on the keyboard than with a mouse. Drag-and-drop is the
 * discoverable path; this is the fast one.
 */
export function useDispatchHotkeys({
  enabled,
  orderedMoves,
  selectedMoveId,
  canUndo,
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

      if (event.key === "u" && canUndo) {
        event.preventDefault();
        onUndo();
        return;
      }
      if (event.key === "t" || event.key === "T") {
        onToday();
        return;
      }
      if (event.key === "ArrowLeft") {
        event.preventDefault();
        onShiftWindow(-1);
        return;
      }
      if (event.key === "ArrowRight") {
        event.preventDefault();
        onShiftWindow(1);
        return;
      }
      if (event.key === "Escape") {
        onClearSelection();
        return;
      }
      if (event.key === "j" || event.key === "k") {
        if (orderedMoves.length === 0) return;
        event.preventDefault();
        const index = orderedMoves.findIndex((move) => move.moveId === selectedMoveId);
        const delta = event.key === "j" ? 1 : -1;
        const next =
          index === -1 ? 0 : Math.min(Math.max(index + delta, 0), orderedMoves.length - 1);
        onSelectMove(orderedMoves[next].moveId);
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
    selectedMoveId,
  ]);
}
