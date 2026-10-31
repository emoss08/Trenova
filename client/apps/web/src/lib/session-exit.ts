import { leaveRegardlessOfWork } from "@/lib/reload-guards";
import { clearSessionCache } from "@/lib/session-cache";
import { announceSignedOut } from "@trenova/shared/lib/session-context";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";

type SessionExit = {
  /** Where to load fresh; null stays on the current page. */
  destination: string | null;
  /**
   * Whether to leave past the browser's prompt for unsaved edits. True when the session was
   * ended on purpose (a sign-out, or someone else signing in, in another tab): the person
   * chose it, and on a shared computer the prompt would only keep the last one's screen up.
   * False when the server ended it under someone mid-work, who may want to copy what they
   * typed before it goes.
   */
  pastUnsavedWork: boolean;
};

/**
 * Leaves a session this tab no longer holds. Nothing of it may stay in memory or on disk for
 * whoever uses the browser next, so the auth state and cache are cleared before the page is
 * loaded fresh at the destination.
 */
export async function leaveEndedSession({
  destination,
  pastUnsavedWork,
}: SessionExit): Promise<void> {
  useAuthStore.getState().clearAuth();
  await clearSessionCache();
  if (destination === null) {
    return;
  }
  if (pastUnsavedWork) {
    leaveRegardlessOfWork(destination);
  } else {
    window.location.assign(destination);
  }
}

/**
 * Ends a session the server has ended (a 401 on a request, or the live stream closed because
 * its session was signed out, revoked or expired). The session is gone for every tab, so they
 * are told rather than left to find out on their next request; the person still at this
 * screen keeps the browser's prompt for unsaved edits.
 */
export function leaveExpiredSession(): void {
  if (!useAuthStore.getState().isAuthenticated) {
    return;
  }

  announceSignedOut();
  void leaveEndedSession({
    destination:
      window.location.pathname === "/login" ? null : signInPathAfter("/login", "expired"),
    pastUnsavedWork: false,
  });
}
