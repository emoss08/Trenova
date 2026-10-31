import { leaveEndedSession } from "@/lib/session-exit";
import { useSessionContextStore } from "@/stores/session-context-store";
import {
  setSessionContextStatusHandler,
  type SessionContextStatus,
} from "@trenova/shared/lib/session-context";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";

/**
 * What the tab does when the shared session changes under it. An organization switch puts
 * the reload prompt in front of the page, since the person may still want what is on it. A
 * sign-out, or someone else signing in, leaves at once: the screen belongs to a person who is
 * no longer signed in, and on a shared computer the next one must not see it.
 */
export function handleSessionContextStatus(status: SessionContextStatus): void {
  const { setOrganizationChanged } = useSessionContextStore.getState();
  switch (status) {
    case "current":
      setOrganizationChanged(false);
      return;
    case "organization-changed":
      setOrganizationChanged(true);
      return;
    case "user-changed":
      void leaveEndedSession({ destination: "/", pastUnsavedWork: true });
      return;
    case "signed-out":
      void leaveEndedSession({
        destination: signInPathAfter("/login", "elsewhere"),
        pastUnsavedWork: true,
      });
      return;
  }
}

export function installSessionContextHandler(): void {
  setSessionContextStatusHandler(handleSessionContextStatus);
}
