import { translate } from "@trenova/shared/i18n/runtime";
import type { BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { toTitleCase } from "@trenova/shared/lib/utils";

export const HOUR_MS = 60 * 60 * 1000;

export function toDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function timeAgo(unixSeconds: number, nowMs: number = Date.now()): string {
  const seconds = Math.max(0, Math.floor(nowMs / 1000) - unixSeconds);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function gaugeTone(remainingMs: number, defaultTone: RingGaugeTone): RingGaugeTone {
  if (remainingMs < HOUR_MS) return "critical";
  if (remainingMs < 2 * HOUR_MS) return "warning";
  return defaultTone;
}

export type DutyStatusInfo = { label: string; variant: BadgeVariant };

const dutyStatuses: Record<string, DutyStatusInfo> = {
  driving: { label: "Driving", variant: "accent-emerald" },
  onDuty: { label: "On Duty", variant: "accent-amber" },
  offDuty: { label: "Off Duty", variant: "accent-slate" },
  sleeperBed: { label: "Sleeper Berth", variant: "accent-violet" },
  yardMove: { label: "Yard Move", variant: "accent-sky" },
  personalConveyance: { label: "Personal Conveyance", variant: "accent-teal" },
};

export function dutyStatusInfo(status: string | null | undefined): DutyStatusInfo {
  if (!status) return { label: translate("Unknown"), variant: "neutral" };
  return dutyStatuses[status] ?? { label: toTitleCase(status), variant: "neutral" };
}
