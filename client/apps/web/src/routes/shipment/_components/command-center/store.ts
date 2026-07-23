import { createSelectors } from "@trenova/shared/lib/utils";
import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import type { TimelineDensity } from "./timeline/constants";

/**
 * Zustand store for ephemeral, cross-component UI state that doesn't belong in
 * the URL: `highlightId` (the row currently pointed at by the table cursor or
 * a map pin) and the dispatch timeline's workstation state — which driver rows
 * are collapsed (ephemeral, the roster changes daily) and the preferred row
 * density (persisted, it's a personal setting). Saved-view selection, filter
 * chips, view mode, expansion, and pagination all live in the URL via
 * `useCommandCenterUrl`.
 */
interface CommandCenterState {
  highlightId: string | null;
  setHighlightId: (id: string | null) => void;
  collapsedRowKeys: Record<string, true>;
  toggleRowCollapsed: (key: string) => void;
  setCollapsedRowKeys: (keys: Record<string, true>) => void;
  timelineDensity: TimelineDensity;
  setTimelineDensity: (density: TimelineDensity) => void;
}

const baseStore = create<CommandCenterState>()(
  devtools(
    persist(
      (set) => ({
        highlightId: null,
        setHighlightId: (id) => set({ highlightId: id }),
        collapsedRowKeys: {},
        toggleRowCollapsed: (key) =>
          set((state) => ({
            collapsedRowKeys: state.collapsedRowKeys[key]
              ? Object.fromEntries(
                  Object.entries(state.collapsedRowKeys).filter(([existing]) => existing !== key),
                )
              : { ...state.collapsedRowKeys, [key]: true },
          })),
        setCollapsedRowKeys: (keys) => set({ collapsedRowKeys: keys }),
        timelineDensity: "comfortable",
        setTimelineDensity: (density) => set({ timelineDensity: density }),
      }),
      {
        name: "cc-timeline-prefs",
        partialize: (state) => ({ timelineDensity: state.timelineDensity }),
      },
    ),
  ),
);

export const useCommandCenterStore = createSelectors(baseStore);
