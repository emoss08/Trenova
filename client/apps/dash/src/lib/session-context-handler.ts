import {
  setSessionContextStatusHandler,
  type SessionContextStatus,
} from "@trenova/shared/lib/session-context";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";

function leaveDashSession(destination: string): void {
  useAuthStore.getState().clearAuth();
  window.location.assign(destination);
}

/**
 * What a portal tab does when the shared session changes under it: it shares the web app's
 * cookie, so a sign-out, someone else signing in or an organization switch in any tab ends
 * what this one shows. The portal keeps its records only in memory, so loading the page
 * fresh is all the teardown it needs.
 */
export function handleDashSessionContextStatus(status: SessionContextStatus): void {
  if (status === "current") {
    return;
  }
  leaveDashSession(status === "signed-out" ? signInPathAfter("/dash/login", "elsewhere") : "/dash");
}

/** The server ended the session behind the portal's live stream. */
export function leaveDashExpiredSession(): void {
  leaveDashSession(signInPathAfter("/dash/login", "expired"));
}

export function installDashSessionContextHandler(): void {
  setSessionContextStatusHandler(handleDashSessionContextStatus);
}
