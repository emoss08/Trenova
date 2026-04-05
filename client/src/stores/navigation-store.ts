import type { ModuleId } from "@/config/navigation.types";
import { create } from "zustand";
import { persist } from "zustand/middleware";

type PanelView = "module" | "favorites";

interface NavigationState {
  sidebarOpen: boolean;
  mobileNavOpen: boolean;
  activeModuleId: ModuleId | null;
  modulePanelCollapsed: boolean;
  panelView: PanelView;

  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setMobileNavOpen: (open: boolean) => void;
  setActiveModuleId: (id: ModuleId | null) => void;
  toggleModulePanel: () => void;
  setModulePanelCollapsed: (collapsed: boolean) => void;
  setPanelView: (view: PanelView) => void;
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      mobileNavOpen: false,
      activeModuleId: null,
      modulePanelCollapsed: false,
      panelView: "module" as PanelView,

      toggleSidebar: () => {
        set((state) => ({
          sidebarOpen: !state.sidebarOpen,
        }));
      },

      setSidebarOpen: (open: boolean) => {
        set({ sidebarOpen: open });
      },

      setMobileNavOpen: (open: boolean) => {
        set({ mobileNavOpen: open });
      },

      setActiveModuleId: (id: ModuleId | null) => {
        set({ activeModuleId: id });
      },

      toggleModulePanel: () => {
        set((state) => ({
          modulePanelCollapsed: !state.modulePanelCollapsed,
        }));
      },

      setModulePanelCollapsed: (collapsed: boolean) => {
        set({ modulePanelCollapsed: collapsed });
      },

      setPanelView: (view: PanelView) => {
        set({ panelView: view });
      },
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
        modulePanelCollapsed: state.modulePanelCollapsed,
      }),
    },
  ),
);
