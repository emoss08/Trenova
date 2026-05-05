import { createSelectors } from "@/lib/utils";
import { create } from "zustand";
import { devtools } from "zustand/middleware";

/**
 * Zustand store for ephemeral, cross-component UI state that doesn't belong in
 * the URL. Right now that's only `highlightId` — the row currently being
 * pointed at by either the table cursor or (in Phase 3) a map pin. Saved-view
 * selection, filter chips, view mode, expansion, and pagination all live in
 * the URL via `useCommandCenterUrl`.
 */
interface CommandCenterState {
  highlightId: string | null;
  setHighlightId: (id: string | null) => void;
}

const baseStore = create<CommandCenterState>()(
  devtools((set) => ({
    highlightId: null,
    setHighlightId: (id) => set({ highlightId: id }),
  })),
);

export const useCommandCenterStore = createSelectors(baseStore);
