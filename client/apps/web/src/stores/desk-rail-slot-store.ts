import { create } from "zustand";

/**
 * The part of the Desk's rail a room fills with its own places.
 *
 * The rail belongs to the Desk's frame and a room to the page under it, so
 * the rail offers the element and the room draws into it. A room's folders
 * then sit in the one rail a person already reads, with the room's own state
 * and actions behind them, and no second side panel beside it.
 */
interface DeskRailSlotState {
  slot: HTMLElement | null;
  setSlot: (slot: HTMLElement | null) => void;
}

export const useDeskRailSlotStore = create<DeskRailSlotState>((set) => ({
  slot: null,
  setSlot: (slot) => set({ slot }),
}));
