import { getPermissionManifest, getPermissionVersion } from "@trenova/shared/lib/permission-api";
import type { OperationType, PermissionManifest } from "@trenova/shared/types/permission";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface PermissionState {
  manifest: PermissionManifest | null;
  isLoading: boolean;
  lastFetched: number | null;

  fetchManifest: () => Promise<void>;
  checkForUpdates: (expectedOrgId?: string) => Promise<boolean>;
  hasPermission: (resource: string, operation: OperationType) => boolean;
  hasAnyPermission: (resource: string, operations: OperationType[]) => boolean;
  hasAllPermissions: (resource: string, operations: OperationType[]) => boolean;
  canAccessRoute: (route: string) => boolean;
  clearPermissions: () => void;
}

const CACHE_DURATION_MS = 5 * 60 * 1000;

export const usePermissionStore = create<PermissionState>()(
  persist(
    (set, get) => ({
      manifest: null,
      isLoading: false,
      lastFetched: null,

      fetchManifest: async () => {
        set({ isLoading: true });
        try {
          const manifest = await getPermissionManifest();
          set({
            manifest,
            isLoading: false,
            lastFetched: Date.now(),
          });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      checkForUpdates: async (expectedOrgId?: string) => {
        const { manifest, lastFetched } = get();

        if (!manifest || !lastFetched) {
          await get().fetchManifest();
          return true;
        }

        if (expectedOrgId && manifest.organizationId !== expectedOrgId) {
          await get().fetchManifest();
          return true;
        }

        if (Date.now() - lastFetched < CACHE_DURATION_MS) {
          return false;
        }

        try {
          const version = await getPermissionVersion();
          if (version.checksum !== manifest.checksum) {
            await get().fetchManifest();
            return true;
          }
          set({ lastFetched: Date.now() });
          return false;
        } catch {
          return false;
        }
      },

      hasPermission: (resource: string, operation: OperationType): boolean => {
        const { manifest } = get();

        if (!manifest) {
          return false;
        }

        // Own-property check, not an undefined check. permissions arrives from
        // JSON.parse, so it inherits Object.prototype: "constructor" and "__proto__"
        // resolve to real values and sail past an undefined guard. They currently deny
        // anyway, but only because Function & operation coerces to 0 — the guard denies
        // by accident rather than by decision, and would flip the moment the bitmask
        // became a truthiness test.
        if (!Object.hasOwn(manifest.permissions, resource)) {
          return false;
        }

        return (manifest.permissions[resource] & operation) !== 0;
      },

      hasAnyPermission: (resource: string, operations: OperationType[]): boolean => {
        const { hasPermission } = get();
        return operations.some((op) => hasPermission(resource, op));
      },

      hasAllPermissions: (resource: string, operations: OperationType[]): boolean => {
        const { hasPermission } = get();
        return operations.every((op) => hasPermission(resource, op));
      },

      canAccessRoute: (route: string): boolean => {
        const { manifest } = get();

        if (!manifest) {
          return false;
        }

        // Same reasoning as hasPermission: `?? false` lets an inherited property through,
        // and routeAccess["constructor"] is truthy.
        if (!Object.hasOwn(manifest.routeAccess, route)) {
          return false;
        }

        return manifest.routeAccess[route] ?? false;
      },

      clearPermissions: () => {
        set({ manifest: null, lastFetched: null, isLoading: false });
      },
    }),
    {
      name: "permission-storage",
      partialize: (state) => ({
        manifest: state.manifest,
        lastFetched: state.lastFetched,
      }),
    },
  ),
);
