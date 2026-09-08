import { safetyRatingLabel, safetyRatingTone } from "./csa";
import { trainingHealthMeta } from "./training";
import { workerTrainingHealthSchema } from "../types/worker-training-health";

export type WorkerHealthMeta = {
  label: string;
  badgeVariant: "active" | "inactive" | "warning" | "secondary";
  /** Colour for a status dot; the only colour a dense roster row carries. */
  dotClass: string;
  /** Whether the state is the one a manager wants to see. */
  good: boolean;
};

const DOT_BY_VARIANT: Record<WorkerHealthMeta["badgeVariant"], string> = {
  active: "bg-green-500",
  inactive: "bg-red-500",
  warning: "bg-amber-500",
  secondary: "bg-muted-foreground/60",
};

export const COMPLIANCE_STATUS_LABELS: Record<string, string> = {
  Compliant: "Compliant",
  NonCompliant: "Non-compliant",
  Pending: "Pending",
};

export function complianceStatusLabel(value: string): string {
  return COMPLIANCE_STATUS_LABELS[value] ?? value;
}

export function complianceStatusMeta(value: string): WorkerHealthMeta {
  switch (value) {
    case "Compliant":
      return {
        label: complianceStatusLabel(value),
        badgeVariant: "active",
        dotClass: DOT_BY_VARIANT.active,
        good: true,
      };
    case "NonCompliant":
      return {
        label: complianceStatusLabel(value),
        badgeVariant: "inactive",
        dotClass: DOT_BY_VARIANT.inactive,
        good: false,
      };
    case "Pending":
      return {
        label: complianceStatusLabel(value),
        badgeVariant: "warning",
        dotClass: DOT_BY_VARIANT.warning,
        good: false,
      };
    default:
      return {
        label: complianceStatusLabel(value),
        badgeVariant: "secondary",
        dotClass: DOT_BY_VARIANT.secondary,
        good: false,
      };
  }
}

export function safetyRatingMeta(value: string): WorkerHealthMeta {
  const badgeVariant = safetyRatingTone(value);
  return {
    label: safetyRatingLabel(value),
    badgeVariant,
    dotClass: DOT_BY_VARIANT[badgeVariant],
    good: value === "Excellent" || value === "Good",
  };
}

/**
 * Training health arrives as a plain string on roster-style queries. An
 * unknown value reads as "Missing" rather than crashing the row, matching the
 * typed helper's own fallback.
 */
export function trainingHealthMetaOf(value: string): WorkerHealthMeta & { blocks: boolean } {
  const parsed = workerTrainingHealthSchema.safeParse(value);
  const meta = trainingHealthMeta(parsed.success ? parsed.data : "Missing");
  return {
    label: meta.label,
    badgeVariant: meta.badgeVariant,
    dotClass: meta.dotClass,
    good: parsed.success && parsed.data === "Current",
    blocks: meta.blocks,
  };
}
