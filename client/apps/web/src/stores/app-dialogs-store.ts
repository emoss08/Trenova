import { create } from "zustand";

/**
 * The app-wide dialogs something other than their own trigger can open: the
 * command palette runs "Keyboard shortcuts" and "Settings" as commands, so
 * their open state cannot live inside the menu that used to own it.
 */
export type AppDialog = "settings" | "shortcuts" | "notifications";

interface AppDialogsState {
  active: AppDialog | null;
  openDialog: (dialog: AppDialog) => void;
  setDialogOpen: (dialog: AppDialog, open: boolean) => void;
  toggleDialog: (dialog: AppDialog) => void;
}

export const useAppDialogsStore = create<AppDialogsState>()((set) => ({
  active: null,
  openDialog: (dialog) => set({ active: dialog }),
  setDialogOpen: (dialog, open) =>
    set((state) => {
      if (open) {
        return { active: dialog };
      }
      return state.active === dialog ? { active: null } : state;
    }),
  toggleDialog: (dialog) => set((state) => ({ active: state.active === dialog ? null : dialog })),
}));

export function useAppDialogOpen(dialog: AppDialog): boolean {
  return useAppDialogsStore((state) => state.active === dialog);
}
