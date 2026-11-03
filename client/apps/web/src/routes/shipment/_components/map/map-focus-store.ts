import { create } from "zustand";

/** What the map's detail card shows: one shipment, one weather alert, or nothing. */
export type MapFocus = { kind: "shipment"; id: string } | { kind: "alert"; id: string } | null;

/** A set of shipments the rail is narrowed to, picked from a cluster or a lane. */
export type MapScope = { ids: string[]; label: string };

export type RailTab = "shipments" | "alerts";

/**
 * The live map's selection, shared by the rail, the marks, the alert polygons
 * and the detail card. Hover stays in the board store (`highlightId`), which
 * the table reads too.
 */
interface MapFocusState {
  focus: MapFocus;
  alertHoverId: string | null;
  scope: MapScope | null;
  railTab: RailTab;
  toggleShipment: (id: string) => void;
  toggleAlert: (id: string) => void;
  clearFocus: () => void;
  setAlertHoverId: (id: string | null) => void;
  setScope: (scope: MapScope | null) => void;
  setRailTab: (tab: RailTab) => void;
  reset: () => void;
}

const INITIAL = {
  focus: null,
  alertHoverId: null,
  scope: null,
  railTab: "shipments",
} satisfies Partial<MapFocusState>;

export const useMapFocusStore = create<MapFocusState>()((set) => ({
  ...INITIAL,
  toggleShipment: (id) =>
    set((state) => ({
      focus:
        state.focus?.kind === "shipment" && state.focus.id === id ? null : { kind: "shipment", id },
    })),
  toggleAlert: (id) =>
    set((state) => ({
      focus: state.focus?.kind === "alert" && state.focus.id === id ? null : { kind: "alert", id },
    })),
  clearFocus: () => set({ focus: null }),
  setAlertHoverId: (id) => set({ alertHoverId: id }),
  setScope: (scope) => set({ scope, railTab: "shipments" }),
  setRailTab: (railTab) => set({ railTab }),
  reset: () => set(INITIAL),
}));
