import { releaseTurnReaders } from "@/components/assistant/turn-readers";
import { clearSessionCache } from "@/lib/session-cache";
import { signInPathAfter, type SignOutNotice } from "@trenova/shared/lib/sign-out-notice";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback } from "react";
import { useNavigate } from "react-router";

/**
 * Ends the session the one way it may end. Any reply still being read is let
 * go first: the server stops the person's replies as it signs them out, and a
 * reader still open would take that for a dropped connection and keep
 * reattaching. The device is cleared and the sign-in page shown even when the
 * server cannot be told, so a failed request never leaves the last session's
 * screen standing; that failure still reaches the caller. `notice` tells the
 * sign-in page why the person is there when they did not choose to be.
 */
export function useSignOut(): (options?: { notice?: SignOutNotice }) => Promise<void> {
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  return useCallback(
    async ({ notice }: { notice?: SignOutNotice } = {}) => {
      releaseTurnReaders();
      try {
        await logout();
      } finally {
        // Cached queries belong to the session that just ended, in memory and on disk; the
        // next sign-in must not see the last organization's plan, usage or records.
        await clearSessionCache();
        void navigate(signInPathAfter("/login", notice));
      }
    },
    [logout, navigate],
  );
}
