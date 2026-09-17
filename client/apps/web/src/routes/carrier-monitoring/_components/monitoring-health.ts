import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import type { CarrierIntelMonitoringStatus } from "@/lib/graphql/carrier-intel-settings";

export type MonitoringHealth =
  | { state: "disconnected"; lastSuccessAt: number | null }
  | {
      state: "paused";
      pausedReason: string | null;
      pausedAt: number | null;
      lastSuccessAt: number | null;
    }
  | { state: "failing"; lastError: string; lastSuccessAt: number | null }
  | { state: "healthy"; lastSuccessAt: number | null };

export function monitoringHealth(status: CarrierIntelMonitoringStatus): MonitoringHealth {
  let lastSuccessAt: number | null = null;
  for (const feed of status.feeds) {
    if (feed.lastSuccessAt && (lastSuccessAt === null || feed.lastSuccessAt > lastSuccessAt)) {
      lastSuccessAt = feed.lastSuccessAt;
    }
  }

  if (!status.provider.configured) {
    return { state: "disconnected", lastSuccessAt };
  }

  const paused = status.feeds.find((feed) => Boolean(feed.pausedReason));
  if (paused) {
    return {
      state: "paused",
      pausedReason: paused.pausedReason ?? null,
      pausedAt: paused.pausedAt ?? null,
      lastSuccessAt,
    };
  }

  const failing = status.feeds.find((feed) => Boolean(feed.lastError) && feed.failureCount > 0);
  if (failing?.lastError) {
    return { state: "failing", lastError: failing.lastError, lastSuccessAt };
  }

  return { state: "healthy", lastSuccessAt };
}

export function integrationSettingsPath(provider: string | null | undefined): string {
  if (!provider) {
    return CARRIER_INTEL_INTEGRATIONS_PATH;
  }
  return `${CARRIER_INTEL_INTEGRATIONS_PATH}&type=${encodeURIComponent(provider)}`;
}
