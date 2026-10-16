import { create } from "zustand";

/**
 * The shipment a pointer is resting on, shared between the table, the map
 * pins and the route overlay. It is too fleeting for the URL.
 */
interface ShipmentBoardState {
  highlightId: string | null;
  setHighlightId: (id: string | null) => void;
}

export const useShipmentBoardStore = create<ShipmentBoardState>()((set) => ({
  highlightId: null,
  setHighlightId: (id) => set({ highlightId: id }),
}));
