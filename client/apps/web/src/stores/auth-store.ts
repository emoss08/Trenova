import { ApiRequestError, clearCsrfToken } from "@/lib/api";
import { realtimeService } from "@/services/realtime";
import { userService } from "@/services/user";
import { authService } from "@/services/auth";
import { usePermissionStore } from "@/stores/permission-store";
import type { LoginRequest, User } from "@/types/user";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;

  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<boolean>;
  setUser: (user: User | null) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isLoading: false,
      isAuthenticated: false,

      login: async (credentials: LoginRequest) => {
        set({ isLoading: true });
        try {
          const response = await authService.login(credentials);
          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
          });
          usePermissionStore.getState().fetchManifest().catch(console.error);
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: async () => {
        try {
          await authService.logout();
        } finally {
          realtimeService.safeClose();
          clearCsrfToken();
          set({ user: null, isAuthenticated: false });
          usePermissionStore.getState().clearPermissions();
        }
      },

      checkAuth: async () => {
        try {
          const freshUser = await userService.currentUser();
          set({ user: freshUser, isAuthenticated: true });
          usePermissionStore
            .getState()
            .checkForUpdates(freshUser.currentOrganizationId)
            .catch(console.error);
          return true;
        } catch (error) {
          if (error instanceof ApiRequestError && error.status === 401) {
            set({ user: null, isAuthenticated: false });
            clearCsrfToken();
            usePermissionStore.getState().clearPermissions();
          }
          return false;
        }
      },

      setUser: (user: User | null) => {
        set({ user, isAuthenticated: !!user });
      },

      clearAuth: () => {
        clearCsrfToken();
        set({ user: null, isAuthenticated: false, isLoading: false });
        usePermissionStore.getState().clearPermissions();
      },
    }),
    {
      name: "auth-storage",
      partialize: (state) => ({ user: state.user }),
    },
  ),
);
