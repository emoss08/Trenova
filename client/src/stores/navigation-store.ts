import { create } from "zustand";
import { persist } from "zustand/middleware";

interface NavigationState {
  sidebarOpen: boolean;
  mobileNavOpen: boolean;

  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setMobileNavOpen: (open: boolean) => void;
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      mobileNavOpen: false,

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
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
      }),
    },
  ),
);
