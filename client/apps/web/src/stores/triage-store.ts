import { create } from "zustand";
import { persist } from "zustand/middleware";

/** What a list row shows under the subject: the next step, or the first words of the message. */
export type TriageRowMode = "next" | "preview";

/**
 * Something drawn over Triage rather than in it. Each kind is opened through
 * `openOverlay` by whatever asked for it — a key, a button, a menu item — and
 * rendered by the page, so the same dialog opens the same way from anywhere.
 */
export type TriageOverlay = { kind: "link"; messageId: string } | { kind: "shortcuts" };

/**
 * What the open reader can do that a key asks for. The reader registers these
 * while it is mounted (`useRegisterReaderCommands`); a key with nothing
 * registered does nothing.
 */
export type ReaderCommands = {
  /** ↵: do the next open step of the open message. */
  doNextStep?: () => void;
  /** R: start or open the reply. */
  reply?: () => void;
};

/** How long a new arrival keeps its entry treatment. */
export const ARRIVAL_MS = 4000;

type TriageState = {
  /** Messages picked for a bulk change. A store slice, so picking one re-renders one row. */
  selected: ReadonlySet<string>;
  /** Where a shift-click range starts. */
  selectionAnchor: string | null;
  /** Messages animating out of the folder before the cache lets them go. */
  leaving: ReadonlySet<string>;
  /** Messages that arrived while the folder was open, for their entry animation. */
  arrivals: ReadonlySet<string>;
  overlay: TriageOverlay | null;
  /** Whether the actions panel is showing beside the open message. */
  panelOpen: boolean;
  /** Whether the folder rail is folded away. Kept per viewer. */
  railCollapsed: boolean;
  /** What rows show under the subject. Kept per viewer. */
  rowMode: TriageRowMode;
  readerCommands: ReaderCommands;
  /** How many Triage pages are mounted; arrival toasts only show while one is. */
  mounted: number;

  toggleSelected: (id: string) => void;
  selectRange: (orderedIds: readonly string[], toId: string) => void;
  setSelected: (ids: Iterable<string>) => void;
  clearSelection: () => void;
  markLeaving: (ids: readonly string[]) => void;
  clearLeaving: (ids: readonly string[]) => void;
  markArrival: (id: string) => void;
  openOverlay: (overlay: TriageOverlay) => void;
  closeOverlay: () => void;
  setPanelOpen: (open: boolean) => void;
  togglePanel: () => void;
  toggleRail: () => void;
  setRowMode: (mode: TriageRowMode) => void;
  setReaderCommands: (commands: ReaderCommands) => void;
  mount: () => () => void;
};

const EMPTY: ReadonlySet<string> = new Set();

function withAll(set: ReadonlySet<string>, ids: readonly string[]): ReadonlySet<string> {
  const next = new Set(set);
  for (const id of ids) {
    next.add(id);
  }

  return next;
}

function without(set: ReadonlySet<string>, ids: readonly string[]): ReadonlySet<string> {
  if (!ids.some((id) => set.has(id))) {
    return set;
  }
  const next = new Set(set);
  for (const id of ids) {
    next.delete(id);
  }

  return next;
}

/**
 * The ids between the anchor and the clicked row, inclusive, in list order.
 * With no anchor in the list, just the clicked row.
 */
export function rangeBetween(
  orderedIds: readonly string[],
  anchor: string | null,
  toId: string,
): string[] {
  const to = orderedIds.indexOf(toId);
  if (to === -1) {
    return [];
  }
  const from = anchor === null ? -1 : orderedIds.indexOf(anchor);
  if (from === -1) {
    return [toId];
  }

  return orderedIds.slice(Math.min(from, to), Math.max(from, to) + 1);
}

/**
 * Triage's view state that is not in the address: what is picked, what is
 * leaving or arriving, what is drawn over the page, and two viewer
 * preferences. Components read single fields with selectors, so a change to
 * one row's selection or the search box never re-renders the list.
 */
export const useTriageStore = create<TriageState>()(
  persist(
    (set, get) => ({
      selected: EMPTY,
      selectionAnchor: null,
      leaving: EMPTY,
      arrivals: EMPTY,
      overlay: null,
      panelOpen: true,
      railCollapsed: false,
      rowMode: "next",
      readerCommands: {},
      mounted: 0,

      toggleSelected: (id) =>
        set((state) => {
          const next = new Set(state.selected);
          if (next.has(id)) {
            next.delete(id);
          } else {
            next.add(id);
          }
          return { selected: next, selectionAnchor: id };
        }),
      selectRange: (orderedIds, toId) =>
        set((state) => ({
          selected: withAll(state.selected, rangeBetween(orderedIds, state.selectionAnchor, toId)),
          selectionAnchor: toId,
        })),
      setSelected: (ids) => set({ selected: new Set(ids), selectionAnchor: null }),
      clearSelection: () =>
        set((state) =>
          state.selected.size === 0 ? state : { selected: EMPTY, selectionAnchor: null },
        ),
      markLeaving: (ids) => set((state) => ({ leaving: withAll(state.leaving, ids) })),
      clearLeaving: (ids) => set((state) => ({ leaving: without(state.leaving, ids) })),
      markArrival: (id) => {
        set((state) => ({ arrivals: withAll(state.arrivals, [id]) }));
        setTimeout(() => set((state) => ({ arrivals: without(state.arrivals, [id]) })), ARRIVAL_MS);
      },
      openOverlay: (overlay) => set({ overlay }),
      closeOverlay: () => set({ overlay: null }),
      setPanelOpen: (panelOpen) => set({ panelOpen }),
      togglePanel: () => set((state) => ({ panelOpen: !state.panelOpen })),
      toggleRail: () => set((state) => ({ railCollapsed: !state.railCollapsed })),
      setRowMode: (rowMode) => set({ rowMode }),
      setReaderCommands: (readerCommands) => set({ readerCommands }),
      mount: () => {
        set((state) => ({ mounted: state.mounted + 1 }));
        return () =>
          set({
            mounted: Math.max(0, get().mounted - 1),
            selected: EMPTY,
            selectionAnchor: null,
            leaving: EMPTY,
            overlay: null,
          });
      },
    }),
    {
      name: "trenova-triage",
      partialize: (state) => ({ railCollapsed: state.railCollapsed, rowMode: state.rowMode }),
    },
  ),
);

export function useIsSelected(id: string): boolean {
  return useTriageStore((state) => state.selected.has(id));
}

export function useIsLeaving(id: string): boolean {
  return useTriageStore((state) => state.leaving.has(id));
}

export function useIsArrival(id: string): boolean {
  return useTriageStore((state) => state.arrivals.has(id));
}
