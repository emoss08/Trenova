import { useSignOut } from "@/hooks/use-sign-out";
import { IdleSessionGuard } from "@trenova/shared/components/session/idle-session-guard";

/** The idle timeout for the TMS: an idle or ended session signs out the usual way. */
export function SessionIdleTimeout() {
  const signOut = useSignOut();

  return (
    <IdleSessionGuard
      onSessionEnd={(notice) => {
        // The server may already have ended the session, so its refusal of the sign-out
        // is expected here; the device is cleared and the sign-in page shown regardless.
        signOut({ notice }).catch(() => {});
      }}
    />
  );
}
