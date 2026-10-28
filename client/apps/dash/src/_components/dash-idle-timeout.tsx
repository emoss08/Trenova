import { useQueryClient } from "@tanstack/react-query";
import { IdleSessionGuard } from "@trenova/shared/components/session/idle-session-guard";
import { signInPathAfter } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useNavigate } from "react-router";

/** The idle timeout for Dash: an idle or ended session goes back to the Dash sign-in. */
export function DashIdleTimeout() {
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return (
    <IdleSessionGuard
      onSessionEnd={(notice) => {
        // The server may already have ended the session, so its refusal of the sign-out
        // is expected; the store is cleared either way and the sign-in page shown.
        void logout()
          .catch(() => {})
          .finally(() => {
            queryClient.clear();
            void navigate(signInPathAfter("/dash/login", notice), { replace: true });
          });
      }}
    />
  );
}
