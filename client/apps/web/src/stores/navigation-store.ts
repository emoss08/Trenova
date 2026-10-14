import type { ModuleId } from "@/config/navigation.types";
import { create } from "zustand";
import { persist } from "zustand/middleware";

export const SIDEBAR_VARIANTS = ["rail", "workspace", "classic"] as const;

export type SidebarVariant = (typeof SIDEBAR_VARIANTS)[number];

export const DEFAULT_SIDEBAR_VARIANT: SidebarVariant = "workspace";

export function isSidebarVariant(value: unknown): value is SidebarVariant {
  return typeof value === "string" && (SIDEBAR_VARIANTS as readonly string[]).includes(value);
}

interface NavigationState {
  activeModuleId: ModuleId | null;
  sidebarCollapsed: boolean;
  /**
   * Whether the sidebar's sheet is out on a phone, where the sidebar never
   * sits beside the page. It starts closed on every visit and is not kept.
   */
  compactSidebarOpen: boolean;
  sidebarVariant: SidebarVariant;
  /** How wide the rail layout's page panel is, in pixels, between its bounds. */
  railPanelWidth: number;

  setActiveModuleId: (id: ModuleId | null) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setCompactSidebarOpen: (open: boolean) => void;
  /** Folds the sidebar beside the page, or on a phone slides its sheet in or out. */
  toggleSidebarFor: (compact: boolean) => void;
  setSidebarVariant: (variant: SidebarVariant) => void;
  setRailPanelWidth: (width: number) => void;
}

export const RAIL_PANEL_MIN_WIDTH = 200;
export const RAIL_PANEL_MAX_WIDTH = 320;
export const RAIL_PANEL_DEFAULT_WIDTH = 224;

export function clampRailPanelWidth(width: number): number {
  if (!Number.isFinite(width)) {
    return RAIL_PANEL_DEFAULT_WIDTH;
  }
  return Math.min(RAIL_PANEL_MAX_WIDTH, Math.max(RAIL_PANEL_MIN_WIDTH, Math.round(width)));
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      activeModuleId: null,
      sidebarCollapsed: false,
      compactSidebarOpen: false,
      sidebarVariant: DEFAULT_SIDEBAR_VARIANT,
      railPanelWidth: RAIL_PANEL_DEFAULT_WIDTH,

      setActiveModuleId: (id: ModuleId | null) => {
        set({ activeModuleId: id });
      },

      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      setCompactSidebarOpen: (open: boolean) => {
        set({ compactSidebarOpen: open });
      },

      toggleSidebarFor: (compact: boolean) => {
        set((state) =>
          compact
            ? { compactSidebarOpen: !state.compactSidebarOpen }
            : { sidebarCollapsed: !state.sidebarCollapsed },
        );
      },

      setSidebarVariant: (variant: SidebarVariant) => {
        set({ sidebarVariant: variant });
      },

      setRailPanelWidth: (width: number) => {
        set({ railPanelWidth: clampRailPanelWidth(width) });
      },
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        sidebarVariant: state.sidebarVariant,
        railPanelWidth: state.railPanelWidth,
      }),
      merge: (persisted, current) => {
        const stored = (persisted ?? {}) as Partial<NavigationState>;
        return {
          ...current,
          ...stored,
          sidebarVariant: isSidebarVariant(stored.sidebarVariant)
            ? stored.sidebarVariant
            : current.sidebarVariant,
          railPanelWidth: clampRailPanelWidth(stored.railPanelWidth ?? current.railPanelWidth),
        };
      },
    },
  ),
);
