import {
  RESOURCE_EVENT_NAME,
  parseInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { realtimeClient } from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { useEffect, useRef } from "react";
import { isPermissionChangeFor } from "@/lib/permission-realtime";

const POLLING_INTERVAL_MS = 5 * 60 * 1000;

export function usePermissionPolling() {
  const checkForUpdates = usePermissionStore((state) => state.checkForUpdates);
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const manifest = usePermissionStore((state) => state.manifest);
  const hasManifest = manifest !== null;
  const currentOrgId = useAuthStore((state) => state.user?.currentOrganizationId);
  const intervalRef = useRef<number | null>(null);

  useEffect(() => {
    if (!manifest) {
      return;
    }

    const poll = async () => {
      try {
        await checkForUpdates(currentOrgId);
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
  }, [manifest, checkForUpdates, currentOrgId]);

  useEffect(() => {
    if (!hasManifest) {
      return undefined;
    }

    return realtimeClient.on(RESOURCE_EVENT_NAME, (data) => {
      if (!isPermissionChangeFor(parseInvalidationEvent(data), currentOrgId)) {
        return;
      }
      void fetchManifest().catch(() => undefined);
    });
  }, [hasManifest, fetchManifest, currentOrgId]);
}
