import { clearCsrfToken } from "@trenova/shared/lib/api";
import { setUserDatePreferences } from "@trenova/shared/lib/date";
import { isSessionExpiry } from "@trenova/shared/lib/error-presentation";
import {
  announceSignedOut,
  sessionContextOf,
  setSessionContext,
} from "@trenova/shared/lib/session-context";
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
import { persist } from "zustand/middleware";

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

/**
 * What a returning tab may know about its person before the server answers: enough to paint
 * the right cached frame, in the right language and clock, and nothing that says who they
 * are. It is all the store writes to the browser, since a shared dispatch-office computer
 * keeps it after the browser closes.
 */
export type RestoredProfile = Pick<
  User,
  "id" | "currentOrganizationId" | "businessUnitId" | "locale" | "timezone" | "timeFormat"
>;

function restoredProfileOf(user: User | null): RestoredProfile | null {
  if (!user) {
    return null;
  }
  const { id, currentOrganizationId, businessUnitId, locale, timezone, timeFormat } = user;
  return { id, currentOrganizationId, businessUnitId, locale, timezone, timeFormat };
}

interface AuthState {
  user: User | null;
  /**
   * The profile restored from storage at startup. Every answer from the server about the
   * session (signed in or out) clears it, so after a sign-out the last person's profile never
   * goes on choosing the cache, language or clock.
   */
  restored: RestoredProfile | null;
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

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      restored: null,
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
            restored: null,
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
            restored: null,
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
          announceSignedOut();
          realtimeClient.disconnect();
          clearCsrfToken();
          set({ user: null, restored: null, isAuthenticated: false });
          usePermissionStore.getState().clearPermissions();
        }
      },

      checkSession: async () => {
        try {
          const freshUser = await userService.currentUser();
          set({ user: freshUser, restored: null, isAuthenticated: true });
          usePermissionStore
            .getState()
            .checkForUpdates(freshUser.currentOrganizationId)
            .catch(console.error);
          return { status: "signed_in" };
        } catch (error) {
          if (!isSessionExpiry(error)) {
            return { status: "unreachable", error };
          }
          set({ user: null, restored: null, isAuthenticated: false });
          clearCsrfToken();
          usePermissionStore.getState().clearPermissions();
          return { status: "signed_out" };
        }
      },

      checkAuth: async () => (await get().checkSession()).status === "signed_in",

      setUser: (user: User | null) => {
        set({ user, restored: null, isAuthenticated: !!user });
      },

      clearAuth: () => {
        clearCsrfToken();
        set({ user: null, restored: null, isAuthenticated: false, isLoading: false });
        usePermissionStore.getState().clearPermissions();
      },
    }),
    {
      name: "auth-storage",
      // Version 0 kept the whole user; migrating drops it, so a browser that stored a person's
      // name, email and phone number before this change no longer holds them.
      version: 1,
      migrate: () => ({ restored: null }),
      partialize: (state) => ({ restored: restoredProfileOf(state.user) ?? state.restored }),
    },
  ),
);

/**
 * Timestamps arrive as UTC epoch seconds and are rendered by
 * `@trenova/shared/lib/date`, which reads the signed-in user's timezone and
 * clock preferences from module state. Mirroring them here — on hydration and
 * on every change — is what keeps a Denver user from reading Eastern times.
 */
const syncUserDatePreferences = (user: Pick<User, "timezone" | "timeFormat"> | null) => {
  setUserDatePreferences({ timezone: user?.timezone, timeFormat: user?.timeFormat });
};

/**
 * The person this tab acts for, as far as it knows: the user once the server has answered,
 * the restored profile before that. Read it for anything that only needs who and where
 * (the cache scope, language, clock), never for what the server must confirm.
 */
export function currentProfile(state: {
  user: User | null;
  restored: RestoredProfile | null;
}): RestoredProfile | null {
  return state.user ?? state.restored;
}

syncUserDatePreferences(currentProfile(useAuthStore.getState()));
useAuthStore.subscribe((state) => syncUserDatePreferences(currentProfile(state)));

/**
 * Every request asserts the session the server last confirmed for this tab: sign-in, an
 * organization switch, a session check. Only the user is a confirmation, and only the server
 * sets it; the profile restored from storage never is, since another tab may have moved the
 * session after writing it. Until the startup session check answers, requests assert
 * nothing.
 */
useAuthStore.subscribe((state, previous) => {
  if (state.user !== previous.user) {
    setSessionContext(sessionContextOf(state.user));
  }
});
