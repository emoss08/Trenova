import type { ModuleId } from "@/config/navigation.types";
import { clampInteger } from "@trenova/shared/lib/utils";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface NavigationState {
  activeModuleId: ModuleId | null;
  sidebarCollapsed: boolean;
  /**
   * Whether the sidebar's sheet is out on a phone, where the sidebar never
   * sits beside the page. It starts closed on every visit and is not kept.
   */
  compactSidebarOpen: boolean;
  /** How wide the panel beside the rail is, in pixels, between its bounds. */
  railPanelWidth: number;

  setActiveModuleId: (id: ModuleId | null) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setCompactSidebarOpen: (open: boolean) => void;
  /** Folds the sidebar beside the page, or on a phone slides its sheet in or out. */
  toggleSidebarFor: (compact: boolean) => void;
  setRailPanelWidth: (width: number) => void;
}

export const RAIL_PANEL_MIN_WIDTH = 200;
export const RAIL_PANEL_MAX_WIDTH = 320;
export const RAIL_PANEL_DEFAULT_WIDTH = 224;

export function clampRailPanelWidth(width: number): number {
  return clampInteger(width, RAIL_PANEL_MIN_WIDTH, RAIL_PANEL_MAX_WIDTH, RAIL_PANEL_DEFAULT_WIDTH);
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      activeModuleId: null,
      sidebarCollapsed: false,
      compactSidebarOpen: false,
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

      setRailPanelWidth: (width: number) => {
        set({ railPanelWidth: clampRailPanelWidth(width) });
      },
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        railPanelWidth: state.railPanelWidth,
      }),
      merge: (persisted, current) => {
        const stored = (persisted ?? {}) as Partial<NavigationState>;
        return {
          ...current,
          sidebarCollapsed: stored.sidebarCollapsed === true,
          railPanelWidth: clampRailPanelWidth(stored.railPanelWidth ?? current.railPanelWidth),
        };
      },
    },
  ),
);
