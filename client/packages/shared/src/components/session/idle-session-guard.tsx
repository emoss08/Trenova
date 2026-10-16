import { IdleTimeoutDialog } from "@trenova/shared/components/session/idle-timeout-dialog";
import { useIdleTimeout } from "@trenova/shared/hooks/use-idle-timeout";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { isSessionExpiry } from "@trenova/shared/lib/error-presentation";
import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";
import { authService } from "@trenova/shared/services/auth";

/**
 * Signs a quiet session out on the server's idle policy, warning first, and keeps a
 * working one alive on the server. Mounted once in each app's signed-in frame; the app
 * decides what ending a session means through `onSessionEnd`, which is handed the notice
 * the sign-in page should show: none when the person chose Sign out on the warning.
 */
export function IdleSessionGuard({
  onSessionEnd,
}: {
  onSessionEnd: (notice: SignOutNotice | undefined) => void;
}) {
  const { config } = usePublicConfig();

  const { phase, deadline, stayActive } = useIdleTimeout({
    policy: config.session,
    onTimeout: () => onSessionEnd("idle"),
    onKeepAlive: () => {
      authService.keepAlive().catch((error: unknown) => {
        // Only a refusal proves the session is gone. A dropped connection or a rate
        // limit leaves it standing, and the next keepalive or request tries again.
        if (isSessionExpiry(error)) {
          onSessionEnd("expired");
        }
      });
    },
  });

  if (phase !== "warning") {
    return null;
  }

  return (
    <IdleTimeoutDialog
      deadline={deadline}
      onStay={stayActive}
      onSignOut={() => onSessionEnd(undefined)}
    />
  );
}
