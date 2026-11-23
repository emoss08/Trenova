import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  WorkPaneApiContext,
  WorkPaneStateContext,
  type WorkPaneApi,
  type WorkPaneEntry,
  type WorkPaneState,
} from "./work-pane-context";

/**
 * One per page, above everything that opens a pane or must know one is open (the record header's
 * Add menu, the tabs, the page's unsaved-changes guard). It keeps one pane at a time in the host.
 */
export function WorkPaneProvider({
  children,
  onDirtyChange,
}: {
  children: ReactNode;
  /** Told whenever the pane showing turns dirty or clean, for a guard on the page itself. */
  onDirtyChange?: (dirty: boolean) => void;
}) {
  const entries = useRef(new Map<string, WorkPaneEntry>());
  const activeRef = useRef<string | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [dirtyId, setDirtyId] = useState<string | null>(null);
  const [host, setHost] = useState<HTMLDivElement | null>(null);

  // Kept by hand, not left to the compiler: every pane registers in an effect keyed on `api`, and a
  // new object (an uncompiled render, a test) would re-register them all and loop on `register`.
  const api = useMemo<WorkPaneApi>(() => {
    const closeThen = (then: () => void) => {
      const id = activeRef.current;
      const entry = id === null ? undefined : entries.current.get(id);
      if (entry) {
        entry.requestClose(then);
      } else {
        then();
      }
    };
    return {
      register: (id, entry) => {
        const previous = activeRef.current;
        entries.current.set(id, entry);
        activeRef.current = id;
        setActiveId(id);
        if (previous !== null && previous !== id) {
          entries.current.get(previous)?.displace();
        }
        return () => {
          if (entries.current.get(id) === entry) entries.current.delete(id);
          if (activeRef.current === id) {
            activeRef.current = null;
            setActiveId(null);
          }
          setDirtyId((current) => (current === id ? null : current));
        };
      },
      setDirty: (id, dirty) =>
        setDirtyId((current) => (dirty ? id : current === id ? null : current)),
      closeThen,
      close: () => closeThen(() => undefined),
      setHost,
    };
  }, []);

  const isDirty = activeId !== null && dirtyId === activeId;
  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  const state: WorkPaneState = { activeId, isDirty, host };

  return (
    <WorkPaneApiContext.Provider value={api}>
      <WorkPaneStateContext.Provider value={state}>{children}</WorkPaneStateContext.Provider>
    </WorkPaneApiContext.Provider>
  );
}
