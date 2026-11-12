import { create } from "zustand";
import { persist } from "zustand/middleware";

/** How a person has set up their Desk. Kept in their browser; nothing here is shared. */
export type DeskSettings = {
  width: "narrow" | "default" | "wide";
  text: "small" | "medium" | "large";
  motion: "full" | "reduced";
  /** The rail beside the page, taking its width, or floating over it when asked for. */
  rail: "pinned" | "floating";
  send: "enter" | "mod";
  refs: "on" | "off";
  start: "today" | "last";
  /** The agent a fresh conversation starts with; empty follows the last one used. */
  agent: string;
  /** Move a question to the agent with the tools for it without asking. */
  autoSwitchAgent: "on" | "off";
  /** With auto switch off: name a better agent, and whether a question sent goes to it. */
  suggestAgent: "switch" | "suggest" | "off";
  sharePage: "on" | "off";
  mentions: "on" | "off";
  slash: "on" | "off";
  mic: "on" | "off";
  presets: "on" | "off";
  drop: "page" | "composer";
  /** The paired computer Scan from Capture starts on; empty asks each time. */
  scanDevice: string;
  /** The scan profile to start from; empty uses the organization's default. */
  scanProfile: string;
  autoOpen: "on" | "off";
  artNotify: "on" | "off";
  celebrate: "confetti" | "subtle" | "off";
  ring: "on" | "off";
};

export const DESK_SETTINGS_DEFAULTS: DeskSettings = {
  width: "default",
  text: "medium",
  motion: "full",
  rail: "pinned",
  send: "enter",
  refs: "on",
  start: "today",
  agent: "",
  autoSwitchAgent: "off",
  suggestAgent: "switch",
  sharePage: "on",
  mentions: "on",
  slash: "on",
  mic: "on",
  presets: "on",
  drop: "page",
  scanDevice: "",
  scanProfile: "",
  autoOpen: "off",
  artNotify: "on",
  celebrate: "confetti",
  ring: "on",
};

/** How many past searches the palette offers back. */
const RECENT_SEARCH_LIMIT = 3;

interface DeskSettingsState {
  settings: DeskSettings;
  /** What this person last searched the Desk for, newest first. */
  recentSearches: string[];
  /** The rail is folded away beside the page. */
  railFolded: boolean;
  rememberSearch: (query: string) => void;
  clearSession: () => void;
  setRailFolded: (folded: boolean) => void;
  set: <K extends keyof DeskSettings>(key: K, value: DeskSettings[K]) => void;
  reset: () => void;
}

export const useDeskSettingsStore = create<DeskSettingsState>()(
  persist(
    (set) => ({
      settings: DESK_SETTINGS_DEFAULTS,
      set: (key, value) => set((state) => ({ settings: { ...state.settings, [key]: value } })),
      reset: () => set({ settings: DESK_SETTINGS_DEFAULTS }),
      recentSearches: [],
      clearSession: () => set({ recentSearches: [] }),
      rememberSearch: (query) =>
        set((state) => ({
          recentSearches: [
            query,
            ...state.recentSearches.filter((past) => past.toLowerCase() !== query.toLowerCase()),
          ].slice(0, RECENT_SEARCH_LIMIT),
        })),
      railFolded: false,
      setRailFolded: (folded) => set({ railFolded: folded }),
    }),
    {
      name: "trenova-desk-settings",
      version: 1,
      partialize: (state) => ({
        settings: state.settings,
        recentSearches: state.recentSearches,
        railFolded: state.railFolded,
      }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<DeskSettingsState> | undefined;
        return {
          ...current,
          settings: {
            ...DESK_SETTINGS_DEFAULTS,
            ...saved?.settings,
            // Pinned unless floating was chosen: a visit from before the
            // setting existed, or a kept value that is neither, is pinned.
            rail: saved?.settings?.rail === "floating" ? "floating" : DESK_SETTINGS_DEFAULTS.rail,
          },
          recentSearches: Array.isArray(saved?.recentSearches) ? saved.recentSearches : [],
          railFolded: saved?.railFolded === true,
        };
      },
    },
  ),
);

/** One Desk setting, read where it applies. */
export function useDeskSetting<K extends keyof DeskSettings>(key: K): DeskSettings[K] {
  return useDeskSettingsStore((state) => state.settings[key]);
}
