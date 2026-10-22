import { create } from "zustand";

/**
 * The app-wide dialogs something other than their own trigger can open: the
 * command palette runs "Keyboard shortcuts" and "Notifications" as commands, so
 * their open state cannot live inside the menu that used to own it.
 */
export type AppDialog = "shortcuts" | "notifications" | "page-access";

interface AppDialogsState {
  active: AppDialog | null;
  /**
   * The dialogs asked for so far: opened once, or about to be (a pointer on
   * the item that opens one). A dialog whose code is read on demand is
   * mounted from then on, so its code arrives before the click and it closes
   * in place.
   */
  prepared: readonly AppDialog[];
  prepareDialog: (dialog: AppDialog) => void;
  openDialog: (dialog: AppDialog) => void;
  setDialogOpen: (dialog: AppDialog, open: boolean) => void;
  toggleDialog: (dialog: AppDialog) => void;
}

function opened(state: AppDialogsState, dialog: AppDialog) {
  return {
    active: dialog,
    prepared: state.prepared.includes(dialog) ? state.prepared : [...state.prepared, dialog],
  };
}

export const useAppDialogsStore = create<AppDialogsState>()((set) => ({
  active: null,
  prepared: [],
  prepareDialog: (dialog) =>
    set((state) =>
      state.prepared.includes(dialog) ? state : { prepared: [...state.prepared, dialog] },
    ),
  openDialog: (dialog) => set((state) => opened(state, dialog)),
  setDialogOpen: (dialog, open) =>
    set((state) => {
      if (open) {
        return opened(state, dialog);
      }
      return state.active === dialog ? { active: null } : state;
    }),
  toggleDialog: (dialog) =>
    set((state) => (state.active === dialog ? { active: null } : opened(state, dialog))),
}));

export function useAppDialogOpen(dialog: AppDialog): boolean {
  return useAppDialogsStore((state) => state.active === dialog);
}

/** Whether a dialog has been asked for, so its code is read and it stays mounted. */
export function useAppDialogPrepared(dialog: AppDialog): boolean {
  return useAppDialogsStore((state) => state.prepared.includes(dialog));
}
