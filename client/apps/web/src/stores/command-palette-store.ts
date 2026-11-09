import type { PaletteScope } from "@/components/command-palette/palette-model";
import { create } from "zustand";

interface CommandPaletteState {
  open: boolean;
  /** The scope the next open lands on; null keeps "All". Taken once the palette opens. */
  requestedScope: PaletteScope | null;
  setOpen: (open: boolean) => void;
  toggleOpen: () => void;
  /** Opens the palette on one scope, such as the Desk's rail search opening Conversations. */
  openOn: (scope: PaletteScope) => void;
  takeRequestedScope: () => PaletteScope | null;
}

export const useCommandPaletteStore = create<CommandPaletteState>()((set, get) => ({
  open: false,
  requestedScope: null,
  setOpen: (open) => set({ open }),
  toggleOpen: () => set((state) => ({ open: !state.open })),
  openOn: (scope) => set({ open: true, requestedScope: scope }),
  takeRequestedScope: () => {
    const scope = get().requestedScope;
    if (scope !== null) {
      set({ requestedScope: null });
    }
    return scope;
  },
}));
