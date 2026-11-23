import { defineLabels } from "@trenova/shared/i18n/labels";
import { toTitleCase } from "@trenova/shared/lib/utils";
import type { BadgeVariant } from "@trenova/shared/types/badge";

/** A driver's duty status as telematics reports it, in words. */
export const DUTY_STATUS_LABELS = defineLabels({
  driving: "Driving",
  onDuty: "On duty",
  offDuty: "Off duty",
  sleeperBed: "Sleeper berth",
  yardMove: "Yard move",
  personalConveyance: "Personal conveyance",
});

export type DutyStatus = keyof typeof DUTY_STATUS_LABELS;

/** A duty status in words; one telematics adds later is title-cased rather than dropped. */
export function dutyStatusLabel(status: string): string {
  return Object.hasOwn(DUTY_STATUS_LABELS, status)
    ? DUTY_STATUS_LABELS[status as DutyStatus]
    : toTitleCase(status);
}

const DUTY_STATUS_VARIANT: Readonly<Record<DutyStatus, BadgeVariant>> = {
  driving: "accent-emerald",
  onDuty: "accent-amber",
  offDuty: "neutral",
  sleeperBed: "accent-violet",
  yardMove: "accent-sky",
  personalConveyance: "accent-teal",
};

/** The badge a duty status wears, wherever it is shown; neutral when unknown or absent. */
export function dutyStatusVariant(status: string | null | undefined): BadgeVariant {
  if (!status) return "neutral";
  return DUTY_STATUS_VARIANT[status as DutyStatus] ?? "neutral";
}
