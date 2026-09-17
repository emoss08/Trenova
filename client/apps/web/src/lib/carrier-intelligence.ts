import type {
  CarrierIntelDepth,
  CarrierIntelEventResolution,
  CarrierIntelEventStatus,
  CarrierIntelRiskLevel,
  CarrierIntelSection,
  CarrierIntelSeverity,
  CarrierIntelVendorState,
} from "@trenova/graphql/generated/graphql";
import { CSA_BASIC_ORDER } from "@trenova/shared/lib/csa";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { compareDecimalStrings, isDecimalString } from "@trenova/shared/types/decimal";
import { z } from "zod";

export type CarrierIntelFindingLike = {
  code: string;
  action: string;
  severity: CarrierIntelSeverity;
};

export type GroupedFindings<T extends CarrierIntelFindingLike> = {
  blockers: T[];
  advisories: T[];
  notices: T[];
};

export type CarrierIntelFreshness = "fresh" | "stale" | "expired";

export const SECONDS_PER_HOUR = 3_600;
export const SECONDS_PER_DAY = 86_400;
export const MAX_OVERRIDE_DAYS = 90;
export const DEFAULT_OVERRIDE_DAYS = 14;
export const MAX_OVERRIDE_SECONDS = MAX_OVERRIDE_DAYS * SECONDS_PER_DAY;
export const CARRIER_INTEL_NOTE_MAX_LENGTH = 2000;
export const DEFAULT_STALE_AFTER_HOURS = 24;
export const DEFAULT_EXPIRED_AFTER_HOURS = 168;

export const CARRIER_INTEL_SEVERITY_RANK: Record<CarrierIntelSeverity, number> = {
  Critical: 0,
  High: 1,
  Medium: 2,
  Low: 3,
  Info: 4,
};

export const CARRIER_INTEL_SEVERITIES = [
  "Critical",
  "High",
  "Medium",
  "Low",
  "Info",
] as const satisfies readonly CarrierIntelSeverity[];

export const CARRIER_INTEL_SECTIONS = [
  "Identity",
  "Authority",
  "Insurance",
  "Safety",
  "Basics",
  "Inspections",
  "Crashes",
  "Fleet",
  "Equipment",
  "Contacts",
  "Operations",
  "ChangeHistory",
  "Network",
  "Lanes",
  "Benchmarks",
] as const satisfies readonly CarrierIntelSection[];

export const CARRIER_INTEL_VENDOR_STATES = [
  "Active",
  "PendingAdd",
  "PendingRemove",
  "Failed",
  "Removed",
  "Unknown",
] as const satisfies readonly CarrierIntelVendorState[];

export const CARRIER_INTEL_RISK_LEVELS = [
  "Low",
  "Moderate",
  "Elevated",
  "High",
  "VeryHigh",
  "Unknown",
] as const satisfies readonly CarrierIntelRiskLevel[];

export const CARRIER_INTEL_EVENT_RESOLUTIONS = [
  "CarrierUpdated",
  "CarrierBlocked",
  "OverrideGranted",
  "NoActionRequired",
  "FalsePositive",
] as const satisfies readonly CarrierIntelEventResolution[];

const PROVIDER_LABELS: Record<string, string> = {
  CarrierOK: "CarrierOk",
  FMCSAQCMobile: "FMCSA QCMobile",
};

export function carrierIntelProviderLabel(provider: string | null | undefined): string {
  if (!provider) {
    return "the provider";
  }
  return PROVIDER_LABELS[provider] ?? provider;
}

export function compareFindings<T extends CarrierIntelFindingLike>(a: T, b: T): number {
  const bySeverity =
    CARRIER_INTEL_SEVERITY_RANK[a.severity] - CARRIER_INTEL_SEVERITY_RANK[b.severity];
  if (bySeverity !== 0) {
    return bySeverity;
  }
  return a.code.localeCompare(b.code);
}

export function groupFindings<T extends CarrierIntelFindingLike>(
  findings: readonly T[],
): GroupedFindings<T> {
  const grouped: GroupedFindings<T> = { blockers: [], advisories: [], notices: [] };
  for (const finding of findings) {
    switch (finding.action) {
      case "Block":
        grouped.blockers.push(finding);
        break;
      case "Warn":
        grouped.advisories.push(finding);
        break;
      case "Notify":
        grouped.notices.push(finding);
        break;
      default:
        break;
    }
  }
  grouped.blockers.sort(compareFindings);
  grouped.advisories.sort(compareFindings);
  grouped.notices.sort(compareFindings);
  return grouped;
}

export function isSectionCovered(
  coverage: readonly CarrierIntelSection[] | null | undefined,
  section: CarrierIntelSection,
): boolean {
  return coverage?.includes(section) ?? false;
}

export function formatOptionalDecimalCurrency(value: string | null | undefined): string | null {
  if (!value || !isDecimalString(value)) {
    return null;
  }
  return formatCurrency(Number(value));
}

export type CoverageStanding = "unknown" | "meets" | "short" | "missing";

export function coverageStanding(
  onFile: string | null | undefined,
  required: string | null | undefined,
): CoverageStanding {
  if (!required || !isDecimalString(required)) {
    return "unknown";
  }
  if (!onFile || !isDecimalString(onFile)) {
    return compareDecimalStrings(required, "0") > 0 ? "missing" : "unknown";
  }
  return compareDecimalStrings(onFile, required) < 0 ? "short" : "meets";
}

export function carrierIntelFreshness({
  effectiveAsOf,
  now,
  staleAfterHours = DEFAULT_STALE_AFTER_HOURS,
  expiredAfterHours = DEFAULT_EXPIRED_AFTER_HOURS,
}: {
  effectiveAsOf: number;
  now: number;
  staleAfterHours?: number;
  expiredAfterHours?: number;
}): CarrierIntelFreshness {
  const age = now - effectiveAsOf;
  if (age > expiredAfterHours * SECONDS_PER_HOUR) {
    return "expired";
  }
  if (age > staleAfterHours * SECONDS_PER_HOUR) {
    return "stale";
  }
  return "fresh";
}

export function probabilityToPercent(value: number): number {
  return value <= 1 ? value * 100 : value;
}

export function sortBasicMeasures<T extends { basic: string }>(measures: readonly T[]): T[] {
  const order: readonly string[] = CSA_BASIC_ORDER;
  const rank = (basic: string) => {
    const index = order.indexOf(basic);
    return index === -1 ? order.length : index;
  };
  return [...measures].sort((a, b) => rank(a.basic) - rank(b.basic));
}

export function fmcsaSafetyRatingTone(
  rating: string,
): "active" | "warning" | "inactive" | "secondary" {
  const normalized = rating.trim().toLowerCase();
  if (normalized === "s" || normalized.startsWith("satisf")) {
    return "active";
  }
  if (normalized === "c" || normalized.startsWith("condition")) {
    return "warning";
  }
  if (normalized === "u" || normalized.startsWith("unsatisf")) {
    return "inactive";
  }
  return "secondary";
}

export function canOverrideFinding(finding: { action: string; overridden: boolean }): boolean {
  return finding.action === "Block" && !finding.overridden;
}

export function canAcknowledgeEvent(status: CarrierIntelEventStatus): boolean {
  return status === "Open";
}

export function canResolveEvent(status: CarrierIntelEventStatus): boolean {
  return status === "Open" || status === "Acknowledged";
}

export const CARRIER_INTEL_DEPTH_CAPABILITY: Record<CarrierIntelDepth, string> = {
  Full: "LookupFull",
  Lite: "LookupLite",
  FMCSA: "LookupFMCSA",
};

export function availableVetDepths(capabilities: readonly string[]): CarrierIntelDepth[] {
  return (["Full", "Lite", "FMCSA"] as const).filter((depth) =>
    capabilities.includes(CARRIER_INTEL_DEPTH_CAPABILITY[depth]),
  );
}

export type CarrierIntelVetCost = {
  amount: number;
  basis: "PerDOTMonth" | "PerMatch" | "Free";
};

const PROVIDER_VET_COSTS: Record<string, Record<CarrierIntelDepth, CarrierIntelVetCost>> = {
  CarrierOK: {
    Full: { amount: 3, basis: "PerDOTMonth" },
    Lite: { amount: 0.5, basis: "PerDOTMonth" },
    FMCSA: { amount: 0.1, basis: "PerMatch" },
  },
};

export function carrierIntelVetCost(
  provider: string | null | undefined,
  depth: CarrierIntelDepth,
): CarrierIntelVetCost {
  if (!provider) {
    return { amount: 0, basis: "Free" };
  }
  return PROVIDER_VET_COSTS[provider]?.[depth] ?? { amount: 0, basis: "Free" };
}

export function createOverrideFormSchema(now: number) {
  return z
    .object({
      reason: z
        .string()
        .trim()
        .min(1, "A reason is required to override a finding")
        .max(CARRIER_INTEL_NOTE_MAX_LENGTH, "Reason cannot exceed 2000 characters"),
      expiresAt: z
        .number({ error: "Choose when the override expires" })
        .int("Choose when the override expires")
        .nullable(),
    })
    .superRefine((values, ctx) => {
      if (values.expiresAt === null) {
        ctx.addIssue({
          code: "custom",
          path: ["expiresAt"],
          message: "Choose when the override expires",
        });
        return;
      }
      if (values.expiresAt <= now) {
        ctx.addIssue({
          code: "custom",
          path: ["expiresAt"],
          message: "An override must expire in the future",
        });
        return;
      }
      if (values.expiresAt - now > MAX_OVERRIDE_SECONDS) {
        ctx.addIssue({
          code: "custom",
          path: ["expiresAt"],
          message: "An override cannot last longer than 90 days",
        });
      }
    });
}

export type OverrideFormValues = z.input<ReturnType<typeof createOverrideFormSchema>>;

export const revokeOverrideFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, "A reason is required to revoke an override")
    .max(CARRIER_INTEL_NOTE_MAX_LENGTH, "Reason cannot exceed 2000 characters"),
});

export type RevokeOverrideFormValues = z.infer<typeof revokeOverrideFormSchema>;

export const reviewNoteFormSchema = z.object({
  note: z
    .string()
    .trim()
    .min(1, "A note is required to mark the carrier reviewed")
    .max(CARRIER_INTEL_NOTE_MAX_LENGTH, "Note cannot exceed 2000 characters"),
});

export type ReviewNoteFormValues = z.infer<typeof reviewNoteFormSchema>;

export const resolveEventFormSchema = z
  .object({
    resolution: z.enum(CARRIER_INTEL_EVENT_RESOLUTIONS, {
      error: "Choose how the event was resolved",
    }),
    note: z
      .string()
      .trim()
      .max(CARRIER_INTEL_NOTE_MAX_LENGTH, "Note cannot exceed 2000 characters"),
  })
  .superRefine((values, ctx) => {
    if (values.resolution === "FalsePositive" && values.note === "") {
      ctx.addIssue({
        code: "custom",
        path: ["note"],
        message: "Explain why this event is a false positive",
      });
    }
  });

export type ResolveEventFormValues = z.infer<typeof resolveEventFormSchema>;

export type CarrierIntelErrorKind =
  | "business"
  | "rate-limit"
  | "validation"
  | "forbidden"
  | "unexpected";

export function classifyCarrierIntelError(error: unknown): CarrierIntelErrorKind {
  if (!(error instanceof GraphQLRequestError)) {
    return "unexpected";
  }
  if (error.isRateLimitError()) {
    return "rate-limit";
  }
  if (error.isAuthorizationError()) {
    return "forbidden";
  }
  if (error.isValidationError()) {
    return "validation";
  }
  if (error.isBusinessError()) {
    return "business";
  }
  return "unexpected";
}
