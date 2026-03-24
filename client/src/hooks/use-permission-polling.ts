import { usePermissionStore } from "@/stores/permission-store";
import { useEffect, useRef } from "react";

const POLLING_INTERVAL_MS = 5 * 60 * 1000;

export function usePermissionPolling() {
  const checkForUpdates = usePermissionStore((state) => state.checkForUpdates);
  const manifest = usePermissionStore((state) => state.manifest);
  const intervalRef = useRef<number | null>(null);

  useEffect(() => {
    if (!manifest) {
      return;
    }

    const poll = async () => {
      try {
        await checkForUpdates();
      } catch {
        // Silently fail - will retry on next interval
      }
    };

    intervalRef.current = window.setInterval(poll, POLLING_INTERVAL_MS);

    return () => {
      if (intervalRef.current) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [manifest, checkForUpdates]);
}
