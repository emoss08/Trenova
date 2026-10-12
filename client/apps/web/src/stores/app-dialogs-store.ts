import { create } from "zustand";

/**
 * The app-wide dialogs something other than their own trigger can open: the
 * command palette runs "Keyboard shortcuts" and "Settings" as commands, so
 * their open state cannot live inside the menu that used to own it.
 */
export type AppDialog = "settings" | "shortcuts" | "notifications";

/** The sections Settings scrolls between; opening with one lands on it. */
export const SETTINGS_SECTIONS = ["profile", "preferences", "notifications", "security"] as const;
export type SettingsSection = (typeof SETTINGS_SECTIONS)[number];

export type OpenDialogOptions = {
  /** Settings only: the section to scroll to once it opens. */
  section?: SettingsSection;
};

interface AppDialogsState {
  active: AppDialog | null;
  /** Where Settings should land the next time it shows; cleared once it has scrolled there. */
  settingsSection: SettingsSection | null;
  openDialog: (dialog: AppDialog, options?: OpenDialogOptions) => void;
  setDialogOpen: (dialog: AppDialog, open: boolean) => void;
  toggleDialog: (dialog: AppDialog) => void;
  clearSettingsSection: () => void;
}

export const useAppDialogsStore = create<AppDialogsState>()((set) => ({
  active: null,
  settingsSection: null,
  openDialog: (dialog, options) =>
    set({
      active: dialog,
      settingsSection: dialog === "settings" ? (options?.section ?? null) : null,
    }),
  setDialogOpen: (dialog, open) =>
    set((state) => {
      if (open) {
        return { active: dialog };
      }
      return state.active === dialog ? { active: null } : state;
    }),
  toggleDialog: (dialog) => set((state) => ({ active: state.active === dialog ? null : dialog })),
  clearSettingsSection: () => set({ settingsSection: null }),
}));

export function useAppDialogOpen(dialog: AppDialog): boolean {
  return useAppDialogsStore((state) => state.active === dialog);
}
