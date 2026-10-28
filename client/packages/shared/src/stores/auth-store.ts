import { clearCsrfToken } from "@trenova/shared/lib/api";
import { setUserDatePreferences } from "@trenova/shared/lib/date";
import { isSessionExpiry } from "@trenova/shared/lib/error-presentation";
import { realtimeClient } from "@trenova/shared/services/realtime";
import { userService } from "@trenova/shared/services/user";
import { authService } from "@trenova/shared/services/auth";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import {
  isMFAChallenge,
  type MFAChallenge,
  type VerifyMFAChallengeRequest,
} from "@trenova/shared/types/mfa";
import type { LoginRequest, User } from "@trenova/shared/types/user";
import { create } from "zustand";

export type LoginOutcome =
  | { status: "signed_in" }
  | { status: "mfa_required"; challenge: MFAChallenge };

/**
 * What a session check learned. Only a 401 proves the session is gone; a rate limit, a
 * server error or a dropped connection says nothing about it, and `unreachable` carries
 * that failure so a caller can present it instead of treating it as a sign-out.
 */
export type SessionCheck =
  | { status: "signed_in" }
  | { status: "signed_out" }
  | { status: "unreachable"; error: unknown };

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;

  login: (credentials: LoginRequest) => Promise<LoginOutcome>;
  verifyMFA: (request: VerifyMFAChallengeRequest) => Promise<void>;
  logout: () => Promise<void>;
  checkSession: () => Promise<SessionCheck>;
  checkAuth: () => Promise<boolean>;
  setUser: (user: User | null) => void;
  clearAuth: () => void;
}

/**
 * Who is signed in. Held in memory only: the person's name, email and profile are never
 * written to the browser's storage, where a shared PC would keep them for whoever sits down
 * next. Every page load learns the session from the server (`checkSession`) before a
 * signed-in route renders, and the language a sign-in page shows in is remembered
 * separately (`trenova.locale`).
 */
export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  isLoading: false,
  isAuthenticated: false,

  login: async (credentials: LoginRequest) => {
    set({ isLoading: true });
    try {
      const response = await authService.login(credentials);
      if (isMFAChallenge(response)) {
        set({ isLoading: false });
        return { status: "mfa_required", challenge: response };
      }
      set({
        user: response.user,
        isAuthenticated: true,
        isLoading: false,
      });
      usePermissionStore.getState().fetchManifest().catch(console.error);
      return { status: "signed_in" };
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },

  verifyMFA: async (request: VerifyMFAChallengeRequest) => {
    set({ isLoading: true });
    try {
      const response = await authService.verifyMFA(request);
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
      realtimeClient.disconnect();
      clearCsrfToken();
      set({ user: null, isAuthenticated: false });
      usePermissionStore.getState().clearPermissions();
    }
  },

  checkSession: async () => {
    try {
      const freshUser = await userService.currentUser();
      set({ user: freshUser, isAuthenticated: true });
      usePermissionStore
        .getState()
        .checkForUpdates(freshUser.currentOrganizationId)
        .catch(console.error);
      return { status: "signed_in" };
    } catch (error) {
      if (!isSessionExpiry(error)) {
        return { status: "unreachable", error };
      }
      set({ user: null, isAuthenticated: false });
      clearCsrfToken();
      usePermissionStore.getState().clearPermissions();
      return { status: "signed_out" };
    }
  },

  checkAuth: async () => (await get().checkSession()).status === "signed_in",

  setUser: (user: User | null) => {
    set({ user, isAuthenticated: !!user });
  },

  clearAuth: () => {
    clearCsrfToken();
    set({ user: null, isAuthenticated: false, isLoading: false });
    usePermissionStore.getState().clearPermissions();
  },
}));

/**
 * Timestamps arrive as UTC epoch seconds and are rendered by
 * `@trenova/shared/lib/date`, which reads the signed-in user's timezone and
 * clock preferences from module state. Mirroring them here on every change is
 * what keeps a Denver user from reading Eastern times.
 */
const syncUserDatePreferences = (user: User | null) => {
  setUserDatePreferences({ timezone: user?.timezone, timeFormat: user?.timeFormat });
};

syncUserDatePreferences(useAuthStore.getState().user);
useAuthStore.subscribe((state) => syncUserDatePreferences(state.user));
