import {
  setSessionContextStatusHandler,
  type SessionContextStatus,
} from "@trenova/shared/lib/session-context";
import { returnPathFor } from "@trenova/shared/lib/router/return-path";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { dashSignInPaths } from "./sign-in-paths";

export function leaveDashSession(destination: string): void {
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

/**
 * The server ended the session behind the portal's live stream. Signing in again brings the
 * driver back to the page they were on.
 */
export function leaveDashExpiredSession(): void {
  const { pathname, search } = window.location;
  leaveDashSession(
    signInPathAfter(
      dashSignInPaths.loginPath,
      "expired",
      returnPathFor(`${pathname}${search}`, dashSignInPaths),
    ),
  );
}

export function installDashSessionContextHandler(): void {
  setSessionContextStatusHandler(handleDashSessionContextStatus);
}
