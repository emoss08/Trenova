import { createSelectors } from "@/lib/utils";
import { create } from "zustand";
import { createJSONStorage, devtools, persist } from "zustand/middleware";

export type RightStackModuleId = "unassigned" | "exceptions" | "hos";

export const ALL_MODULES: readonly RightStackModuleId[] = [
  "unassigned",
  "exceptions",
  "hos",
] as const;

interface RightStackState {
  order: RightStackModuleId[];
  hidden: RightStackModuleId[];
  hide: (id: RightStackModuleId) => void;
  show: (id: RightStackModuleId) => void;
  reorder: (next: RightStackModuleId[]) => void;
}

const baseStore = create<RightStackState>()(
  devtools(
    persist(
      (set, get) => ({
        order: [...ALL_MODULES],
        hidden: [],
        hide: (id) =>
          set({
            hidden: get().hidden.includes(id) ? get().hidden : [...get().hidden, id],
          }),
        show: (id) => set({ hidden: get().hidden.filter((h) => h !== id) }),
        reorder: (next) => set({ order: next }),
      }),
      {
        name: "trenova.rightStack",
        storage: createJSONStorage(() => localStorage),
      },
    ),
  ),
);

export const useRightStackStore = createSelectors(baseStore);
