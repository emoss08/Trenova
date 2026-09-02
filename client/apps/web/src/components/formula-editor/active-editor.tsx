import type { EditorView } from "@codemirror/view";
import { createContext, useCallback, useContext, useMemo, useRef, type ReactNode } from "react";
import { insertSnippet } from "./insert-at-cursor";

type ActiveEditorContextValue = {
  setActive: (view: EditorView | null) => void;
  setPrimary: (view: EditorView | null) => void;
  insert: (text: string, cursorOffset?: number) => boolean;
};

const ActiveEditorContext = createContext<ActiveEditorContextValue | null>(null);

/**
 * Tracks which expression editor the person touched last, so a click in the
 * reference pane lands in the breakdown line they were writing rather than
 * always in the main expression. The primary editor is the fallback when
 * nothing has been focused yet.
 */
export function ActiveEditorProvider({ children }: { children: ReactNode }) {
  const activeRef = useRef<EditorView | null>(null);
  const primaryRef = useRef<EditorView | null>(null);

  const setActive = useCallback((view: EditorView | null) => {
    activeRef.current = view;
  }, []);

  const setPrimary = useCallback((view: EditorView | null) => {
    primaryRef.current = view;
  }, []);

  const insert = useCallback((text: string, cursorOffset?: number) => {
    const target = activeRef.current ?? primaryRef.current;
    if (!target) return false;
    insertSnippet(target, text, cursorOffset);
    return true;
  }, []);

  const value = useMemo(() => ({ setActive, setPrimary, insert }), [setActive, setPrimary, insert]);

  return <ActiveEditorContext.Provider value={value}>{children}</ActiveEditorContext.Provider>;
}

/** Registration hooks for an editor; no-ops outside a provider. */
export function useActiveEditorRegistration() {
  const context = useContext(ActiveEditorContext);
  return {
    setActive: context?.setActive,
    setPrimary: context?.setPrimary,
  };
}

/** Inserts into the last-focused editor (or the primary one). */
export function useActiveEditorInsert(): (text: string, cursorOffset?: number) => boolean {
  const context = useContext(ActiveEditorContext);
  return useCallback(
    (text: string, cursorOffset?: number) => context?.insert(text, cursorOffset) ?? false,
    [context],
  );
}
