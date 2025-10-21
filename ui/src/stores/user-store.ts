import { clearPermissionCache } from "@/lib/loaders";
import type { UserSchema } from "@/lib/schemas/user-schema";
import { create } from "zustand";
import { useShallow } from "zustand/react/shallow";

interface AuthState {
  user: UserSchema | null;
  isAuthenticated: boolean;
  isInitialized: boolean;
  permissions: Set<string>;
  setUser: (user: UserSchema | null) => void;
  setInitialized: (initialized: boolean) => void;
  clearAuth: () => void;
  hasPermission: (resource: string, action: string) => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isInitialized: false,
  permissions: new Set<string>(),
  setUser: (user) => {
    const newPermissions = new Set<string>();
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
  clearAuth: () => {
    clearPermissionCache();
    set({
      user: null,
      isAuthenticated: false,
      isInitialized: false,
      permissions: new Set<string>(),
    });
  },
  hasPermission: (resource: string, action: string) => {
    return get().permissions.has(`${resource}:${action}`);
  },
}));

export const useUser = () => useAuthStore((state) => state.user);

export const useIsAuthenticated = () =>
  useAuthStore((state) => state.isAuthenticated);

export const useIsInitialized = () =>
  useAuthStore((state) => state.isInitialized);

export const useAuthActions = () =>
  useAuthStore(
    useShallow((state) => ({
      setUser: state.setUser,
      setInitialized: state.setInitialized,
      clearAuth: state.clearAuth,
    })),
  );

export const usePermissionCheck = () => {
  const hasPermissionFn = useAuthStore((state) => state.hasPermission);
  const permissionsSet = useAuthStore((state) => state.permissions);
  return { hasPermission: hasPermissionFn, _permissionsSet: permissionsSet };
};

export const useHasPermission = () =>
  useAuthStore((state) => state.hasPermission);
