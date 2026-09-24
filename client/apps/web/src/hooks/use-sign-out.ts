import { releaseTurnReaders } from "@/components/assistant/turn-readers";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback } from "react";
import { useNavigate } from "react-router";

/**
 * Ends the session the one way it may end. Any reply still being read is let
 * go first: the server stops the person's replies as it signs them out, and a
 * reader still open would take that for a dropped connection and keep
 * reattaching.
 */
export function useSignOut(): () => Promise<void> {
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  return useCallback(async () => {
    releaseTurnReaders();
    await logout();
    void navigate("/login");
  }, [logout, navigate]);
}
