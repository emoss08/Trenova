import type { ModuleId } from "@/config/navigation.types";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface NavigationState {
  activeModuleId: ModuleId | null;
  sidebarCollapsed: boolean;
  activitySectionOpen: boolean;

  setActiveModuleId: (id: ModuleId | null) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setActivitySectionOpen: (open: boolean) => void;
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      activeModuleId: null,
      sidebarCollapsed: false,
      activitySectionOpen: true,

      setActiveModuleId: (id: ModuleId | null) => {
        set({ activeModuleId: id });
      },

      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      setActivitySectionOpen: (open: boolean) => {
        set({ activitySectionOpen: open });
      },
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        activitySectionOpen: state.activitySectionOpen,
      }),
    },
  ),
);
