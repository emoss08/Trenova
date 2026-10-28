import { DESK_SHELF_KEYS, type DeskShelfKey } from "@/components/desk-chat/rail/desk-threads";
import { clampInteger } from "@trenova/shared/lib/utils";
import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * How a person has set up their Desk. Kept in their browser: each is a choice about how the
 * Desk looks or behaves, or the id of a computer, profile or agent, and none names a record
 * or a person.
 */
export type DeskSettings = {
  width: "narrow" | "default" | "wide";
  text: "small" | "medium" | "large";
  motion: "full" | "reduced";
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

export const DESK_RAIL_MIN_WIDTH = 200;
export const DESK_RAIL_MAX_WIDTH = 340;
export const DESK_RAIL_DEFAULT_WIDTH = 232;

/** The rail's width in pixels, between its bounds; anything else reads as the default. */
export function clampDeskRailWidth(width: number): number {
  return clampInteger(width, DESK_RAIL_MIN_WIDTH, DESK_RAIL_MAX_WIDTH, DESK_RAIL_DEFAULT_WIDTH);
}

/** How many past searches the palette offers back. */
const RECENT_SEARCH_LIMIT = 3;

interface DeskSettingsState {
  settings: DeskSettings;
  /**
   * What this person last searched the Desk for, newest first. Kept in memory only, since a
   * search names what was searched for, and forgotten when the session ends.
   */
  recentSearches: string[];
  /** The rail's shelves this person has folded away. */
  collapsedShelves: DeskShelfKey[];
  /** How wide this person has dragged the rail, in pixels. */
  railWidth: number;
  /** The rail is folded away beside the page. */
  railFolded: boolean;
  rememberSearch: (query: string) => void;
  forgetSearches: () => void;
  toggleShelf: (key: DeskShelfKey) => void;
  setRailWidth: (width: number) => void;
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
      rememberSearch: (query) =>
        set((state) => ({
          recentSearches: [
            query,
            ...state.recentSearches.filter((past) => past.toLowerCase() !== query.toLowerCase()),
          ].slice(0, RECENT_SEARCH_LIMIT),
        })),
      forgetSearches: () => set({ recentSearches: [] }),
      collapsedShelves: [],
      toggleShelf: (key) =>
        set((state) => ({
          collapsedShelves: state.collapsedShelves.includes(key)
            ? state.collapsedShelves.filter((shelf) => shelf !== key)
            : [...state.collapsedShelves, key],
        })),
      railWidth: DESK_RAIL_DEFAULT_WIDTH,
      setRailWidth: (width) => set({ railWidth: clampDeskRailWidth(width) }),
      railFolded: false,
      setRailFolded: (folded) => set({ railFolded: folded }),
    }),
    {
      name: "trenova-desk-settings",
      version: 1,
      partialize: (state) => ({
        settings: state.settings,
        collapsedShelves: state.collapsedShelves,
        railWidth: state.railWidth,
        railFolded: state.railFolded,
      }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<DeskSettingsState> | undefined;
        return {
          ...current,
          settings: { ...DESK_SETTINGS_DEFAULTS, ...saved?.settings },
          collapsedShelves: Array.isArray(saved?.collapsedShelves)
            ? DESK_SHELF_KEYS.filter((key) => saved.collapsedShelves?.includes(key))
            : [],
          railWidth: clampDeskRailWidth(saved?.railWidth ?? DESK_RAIL_DEFAULT_WIDTH),
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
