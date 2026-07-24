import { createSelectors } from "@trenova/shared/lib/utils";
import { create } from "zustand";
import { createJSONStorage, devtools, persist } from "zustand/middleware";

export type RightStackModuleId = "unassigned" | "exceptions" | "hos" | "certification";

export const ALL_MODULES: readonly RightStackModuleId[] = [
  "unassigned",
  "exceptions",
  "hos",
  "certification",
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
        merge: (persisted, current) => {
          const saved = (persisted as Partial<RightStackState>) ?? {};
          const savedOrder = saved.order ?? [];
          const savedHidden = saved.hidden ?? [];
          const known = new Set<RightStackModuleId>(ALL_MODULES);
          const order = savedOrder.filter((id) => known.has(id));
          for (const id of ALL_MODULES) {
            if (!order.includes(id)) {
              order.push(id);
            }
          }
          return {
            ...current,
            order,
            hidden: savedHidden.filter((id) => known.has(id)),
          };
        },
      },
    ),
  ),
);

export const useRightStackStore = createSelectors(baseStore);
