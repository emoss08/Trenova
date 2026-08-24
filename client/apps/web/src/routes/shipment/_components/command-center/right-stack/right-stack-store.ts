import { createSelectors } from "@trenova/shared/lib/utils";
import {
  hasOrganizationCapability,
  OrganizationCapability,
  type OrganizationCapabilities,
  type OrganizationCapabilityType,
} from "@trenova/shared/types/organization-capability";
import { create } from "zustand";
import { createJSONStorage, devtools, persist } from "zustand/middleware";

export type RightStackModuleId = "unassigned" | "exceptions" | "hos" | "certification";

export const ALL_MODULES: readonly RightStackModuleId[] = [
  "unassigned",
  "exceptions",
  "hos",
  "certification",
] as const;

const MODULE_CAPABILITY: Partial<Record<RightStackModuleId, OrganizationCapabilityType>> = {
  hos: OrganizationCapability.AssetOperations,
  certification: OrganizationCapability.AssetOperations,
};

/**
 * Narrows a stack order — or a hidden list — to the modules the organization can
 * actually use. HOS clocks and unsigned-log counts are readings off employed
 * drivers, so a brokerage has nothing to put in them.
 *
 * The filter is applied where the stack renders rather than written back into
 * the store: a layout preference outlives a capability change, and an
 * organization that switches its own trucks back on should find its arrangement
 * intact rather than silently rewritten.
 */
export function availableRightStackModules(
  ids: readonly RightStackModuleId[],
  capabilities: OrganizationCapabilities,
): RightStackModuleId[] {
  return ids.filter((id) => {
    const capability = MODULE_CAPABILITY[id];
    return !capability || hasOrganizationCapability(capabilities, capability);
  });
}

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
