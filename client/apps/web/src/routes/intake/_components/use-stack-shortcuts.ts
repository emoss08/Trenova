import { useHotkey } from "@tanstack/react-hotkeys";
import { useCallback } from "react";

/** Whether a menu or dialog has the keyboard, which a stack's letters must leave alone. */
function overlayOpen(): boolean {
  return (
    document.querySelector(
      '[role="menu"], [role="dialog"], [role="alertdialog"], [role="listbox"]',
    ) !== null
  );
}

export type StackShortcuts = {
  /** Moves to the next (1) or previous (-1) document. */
  move: (step: 1 | -1) => void;
  fileCurrent: () => void;
  turnCurrent: () => void;
  save: () => void;
};

/**
 * The letters that work a stack from the keyboard: J and K move between its
 * documents, F files the one you are on, R turns its pages a quarter to the
 * right and S saves the split. They stand down while typing, and while a menu
 * or dialog is open.
 */
export function useStackShortcuts(shortcuts: StackShortcuts, enabled: boolean) {
  const guarded = useCallback(
    (run: () => void) => () => {
      if (!overlayOpen()) {
        run();
      }
    },
    [],
  );
  const options = { ignoreInputs: true, preventDefault: true, enabled };

  useHotkey(
    "J",
    guarded(() => shortcuts.move(1)),
    options,
  );
  useHotkey(
    "K",
    guarded(() => shortcuts.move(-1)),
    options,
  );
  useHotkey("F", guarded(shortcuts.fileCurrent), options);
  useHotkey("R", guarded(shortcuts.turnCurrent), options);
  useHotkey("S", guarded(shortcuts.save), options);
}
