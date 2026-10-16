import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { createIdleMonitor, type IdleSnapshot } from "@trenova/shared/lib/idle-monitor";
import { sessionActivityClock } from "@trenova/shared/lib/session-activity";
import type { SessionIdlePolicy } from "@trenova/shared/types/platform";
import { useMemo, useSyncExternalStore } from "react";

export type IdleTimeoutState = IdleSnapshot & { stayActive: () => void };

/**
 * The signed-in tab's place against the idle timeout (see `createIdleMonitor`). The
 * monitor is rebuilt only when the policy's numbers change, so a fresh public-config
 * object with the same values does not restart the countdown.
 */
export function useIdleTimeout({
  policy,
  onTimeout,
  onKeepAlive,
}: {
  policy: SessionIdlePolicy;
  onTimeout: () => void;
  onKeepAlive: () => void;
}): IdleTimeoutState {
  const handleTimeout = useLatestCallback(onTimeout);
  const handleKeepAlive = useLatestCallback(onKeepAlive);
  const { idleTimeoutSeconds, idleWarningSeconds, keepAliveSeconds } = policy;

  const monitor = useMemo(
    () =>
      createIdleMonitor({
        policy: { idleTimeoutSeconds, idleWarningSeconds, keepAliveSeconds },
        clock: sessionActivityClock,
        onTimeout: handleTimeout,
        onKeepAlive: handleKeepAlive,
      }),
    [idleTimeoutSeconds, idleWarningSeconds, keepAliveSeconds, handleTimeout, handleKeepAlive],
  );

  const snapshot = useSyncExternalStore(
    monitor.subscribe,
    monitor.getSnapshot,
    monitor.getSnapshot,
  );

  return { ...snapshot, stayActive: monitor.stayActive };
}
