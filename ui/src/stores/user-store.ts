/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import type { UserSchema } from "@/lib/schemas/user-schema";
import { create } from "zustand";
import { useShallow } from "zustand/react/shallow";

interface AuthState {
  user: UserSchema | null;
  isAuthenticated: boolean;
  isInitialized: boolean;
  // Store permissions as a Set for efficient lookup: "resource:action"
  permissions: Set<string>;
  setUser: (user: UserSchema | null) => void;
  setInitialized: (initialized: boolean) => void;
  clearAuth: () => void;
  // Function to check permission
  hasPermission: (resource: string, action: string) => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isInitialized: false,
  permissions: new Set<string>(),
  setUser: (user) => {
    const newPermissions = new Set<string>();
    if (user && user.roles) {
      user.roles.forEach((role) => {
        // Ensure role and role.permissions exist
        if (role && role.permissions) {
          role.permissions.forEach((permission) => {
            // Ensure permission, resource, and action exist
            if (permission && permission.resource && permission.action) {
              newPermissions.add(`${permission.resource}:${permission.action}`);
            }
          });
        }
      });
    }
    set({
      user,
      isAuthenticated: !!user,
      permissions: newPermissions,
    });
  },
  setInitialized: (initialized) =>
    set({
      isInitialized: initialized,
    }),
  clearAuth: () =>
    set({
      user: null,
      isAuthenticated: false,
      isInitialized: false,
      permissions: new Set<string>(),
    }),
  hasPermission: (resource: string, action: string) => {
    return get().permissions.has(`${resource}:${action}`);
  },
}));

// Optimized selectors to prevent unnecessary re-renders
export const useUser = () => useAuthStore((state) => state.user);

export const useIsAuthenticated = () =>
  useAuthStore((state) => state.isAuthenticated);

export const useIsInitialized = () =>
  useAuthStore((state) => state.isInitialized);

// Use the useShallow hook to memoize the selector for auth actions
export const useAuthActions = () =>
  useAuthStore(
    useShallow((state) => ({
      setUser: state.setUser,
      setInitialized: state.setInitialized,
      clearAuth: state.clearAuth,
    })),
  );

// Selector for checking permissions
export const usePermissionCheck = () => {
  const hasPermissionFn = useAuthStore((state) => state.hasPermission);
  const permissionsSet = useAuthStore((state) => state.permissions); // if direct access to the set is needed
  return { hasPermission: hasPermissionFn, _permissionsSet: permissionsSet }; // _permissionsSet for debugging or advanced use
};

// Convenience hook to get the hasPermission function directly
export const useHasPermission = () =>
  useAuthStore((state) => state.hasPermission);
