import { IdleSessionGuard } from "@trenova/shared/components/session/idle-session-guard";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { leaveDashSession } from "@/lib/session-context-handler";

/**
 * The idle timeout for Dash: an idle or ended session goes back to the Dash sign-in. The page
 * loads fresh, because the portal's records live in memory and the next driver must not get them.
 */
export function DashIdleTimeout() {
  const logout = useAuthStore((state) => state.logout);

  return (
    <IdleSessionGuard
      onSessionEnd={(notice) => {
        // The server may already have ended the session, so its refusal of the sign-out
        // is expected; the store is cleared either way and the sign-in page shown.
        void logout()
          .catch(() => {})
          .finally(() => {
            leaveDashSession(signInPathAfter("/dash/login", notice));
          });
      }}
    />
  );
}
