import type { ModuleId } from "@/config/navigation.types";
import { create } from "zustand";
import { persist } from "zustand/middleware";

export const SIDEBAR_VARIANTS = ["workspace", "classic"] as const;

export type SidebarVariant = (typeof SIDEBAR_VARIANTS)[number];

export const DEFAULT_SIDEBAR_VARIANT: SidebarVariant = "workspace";

export function isSidebarVariant(value: unknown): value is SidebarVariant {
  return typeof value === "string" && (SIDEBAR_VARIANTS as readonly string[]).includes(value);
}

interface NavigationState {
  activeModuleId: ModuleId | null;
  sidebarCollapsed: boolean;
  sidebarVariant: SidebarVariant;

  setActiveModuleId: (id: ModuleId | null) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setSidebarVariant: (variant: SidebarVariant) => void;
}

export const useNavigationStore = create<NavigationState>()(
  persist(
    (set) => ({
      activeModuleId: null,
      sidebarCollapsed: false,
      sidebarVariant: DEFAULT_SIDEBAR_VARIANT,

      setActiveModuleId: (id: ModuleId | null) => {
        set({ activeModuleId: id });
      },

      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      setSidebarVariant: (variant: SidebarVariant) => {
        set({ sidebarVariant: variant });
      },
    }),
    {
      name: "navigation-storage",
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        sidebarVariant: state.sidebarVariant,
      }),
      merge: (persisted, current) => {
        const stored = (persisted ?? {}) as Partial<NavigationState>;
        return {
          ...current,
          ...stored,
          sidebarVariant: isSidebarVariant(stored.sidebarVariant)
            ? stored.sidebarVariant
            : current.sidebarVariant,
        };
      },
    },
  ),
);
