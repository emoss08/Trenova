import type {
  CarrierIntelDepth,
  CarrierIntelEventResolution,
  CarrierIntelEventStatus,
  CarrierIntelNetworkKind,
  CarrierIntelRiskLevel,
  CarrierIntelSection,
  CarrierIntelSeverity,
  CarrierIntelVendorState,
} from "@trenova/graphql/generated/graphql";
import { intlLocale } from "@trenova/shared/i18n/format";
import { CSA_BASIC_ORDER } from "@trenova/shared/lib/csa";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { compareDecimalStrings, isDecimalString } from "@trenova/shared/types/decimal";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
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

export function canOverrideFinding(finding: { action: string; overridden: boolean }): boolean {
  return finding.action === "Block" && !finding.overridden;
}

export type CarrierIntelOverrideState = "active" | "revoked" | "expired";

export function carrierIntelOverrideState(override: {
  active: boolean;
  revokedAt: number | null;
}): CarrierIntelOverrideState {
  if (override.active) {
    return "active";
  }
  return override.revokedAt ? "revoked" : "expired";
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

export type CarrierIntelDepthMerge = {
  held: CarrierIntelDepth;
  heldAt: number;
  fetched: CarrierIntelDepth;
  fetchedAt: number;
};

export function carrierIntelDepthMerge({
  depth,
  depthFetchedAt,
  fetchedDepth,
  fetchedAt,
}: {
  depth: CarrierIntelDepth | null | undefined;
  depthFetchedAt: number | null | undefined;
  fetchedDepth: CarrierIntelDepth | null | undefined;
  fetchedAt: number | null | undefined;
}): CarrierIntelDepthMerge | null {
  if (!depth || !fetchedDepth || depth === fetchedDepth || !depthFetchedAt || !fetchedAt) {
    return null;
  }
  return { held: depth, heldAt: depthFetchedAt, fetched: fetchedDepth, fetchedAt };
}

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

export function isCarrierIntelNotFound(error: unknown): boolean {
  return error instanceof GraphQLRequestError && error.isNotFoundError();
}

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

export const DAYS_PER_YEAR = 365;
export const DAYS_PER_MONTH = 30;

export type CarrierAge = { unit: "year" | "month"; value: number };

export function ageFromDays(days: number | null | undefined): CarrierAge | null {
  if (days === null || days === undefined || !Number.isFinite(days) || days < 0) {
    return null;
  }
  if (days >= DAYS_PER_YEAR) {
    return { unit: "year", value: Math.floor(days / DAYS_PER_YEAR) };
  }
  return { unit: "month", value: Math.floor(days / DAYS_PER_MONTH) };
}

export function cityStateLabel(
  city: string | null | undefined,
  state: string | null | undefined,
): string | null {
  if (city && state) {
    return `${city}, ${state}`;
  }
  return city || state || null;
}

type AuthorityGrantLike = { status: string; ageDays: number | null } | null | undefined;

export function oldestActiveAuthorityAgeDays(
  authority:
    | Pick<NonNullable<CarrierIntelProfile["authority"]>, "common" | "contract" | "broker">
    | null
    | undefined,
): number | null {
  if (!authority) {
    return null;
  }
  let oldest: number | null = null;
  const grants: AuthorityGrantLike[] = [authority.common, authority.contract, authority.broker];
  for (const grant of grants) {
    if (!grant || grant.status !== "Active" || grant.ageDays === null) {
      continue;
    }
    if (oldest === null || grant.ageDays > oldest) {
      oldest = grant.ageDays;
    }
  }
  return oldest;
}

export function profileAuthorityAgeDays(
  profile: Pick<CarrierIntelProfile, "authority" | "identity">,
): number | null {
  return oldestActiveAuthorityAgeDays(profile.authority) ?? profile.identity?.dotAgeDays ?? null;
}

type IntelNetworkLink = NonNullable<NonNullable<CarrierIntelProfile["network"]>["links"]>[number];

const NETWORK_KIND_ORDER: readonly CarrierIntelNetworkKind[] = [
  "EIN",
  "Equipment",
  "Address",
  "Phone",
  "Email",
];

export type LinkedCarrier = {
  dotNumber: string;
  kinds: CarrierIntelNetworkKind[];
};

export function groupNetworkLinks(links: readonly IntelNetworkLink[]): LinkedCarrier[] {
  const byDot = new Map<string, Set<CarrierIntelNetworkKind>>();
  for (const link of links) {
    if (!link.dotNumber) {
      continue;
    }
    const kinds = byDot.get(link.dotNumber) ?? new Set<CarrierIntelNetworkKind>();
    kinds.add(link.kind);
    byDot.set(link.dotNumber, kinds);
  }
  const rank = (kind: CarrierIntelNetworkKind) => NETWORK_KIND_ORDER.indexOf(kind);
  return [...byDot.entries()]
    .map(([dotNumber, kinds]) => ({
      dotNumber,
      kinds: [...kinds].sort((a, b) => rank(a) - rank(b)),
    }))
    .sort(
      (a, b) =>
        rank(a.kinds[0]) - rank(b.kinds[0]) ||
        b.kinds.length - a.kinds.length ||
        a.dotNumber.localeCompare(b.dotNumber),
    );
}

export function joinPresent(
  parts: readonly (string | number | null | undefined)[],
  separator: string,
): string | null {
  const text = parts
    .filter((part) => part !== null && part !== undefined && part !== "")
    .join(separator);
  return text === "" ? null : text;
}

export type IntelValueLabels = {
  yes: string;
  no: string;
  empty: string;
};

export const INTEL_EMPTY_VALUE = "—";

const DEFAULT_INTEL_VALUE_LABELS: IntelValueLabels = {
  yes: "Yes",
  no: "No",
  empty: INTEL_EMPTY_VALUE,
};

const INTEL_MONEY_PATH = /^insurance\.(bipd|cargo|bond)(OnFile|Required)$/;
const INTEL_ENUM_PATH = /^(safety\.(rating|riskScore)|authority\.[A-Za-z]+\.status)$/;
const INTEL_DATE_LEAF = /(At|Date)$/;
const INTEL_AGE_DAYS_LEAF = /(^a|A)geDays$/;
const INTEL_INTEGER = /^-?\d+$/;

function unquoteIntelValue(text: string): string {
  if (text.length >= 2 && text.startsWith('"') && text.endsWith('"')) {
    return text.slice(1, -1);
  }
  return text;
}

function humanizeIntelEnum(value: string): string {
  const words = value.replace(/([a-z0-9])([A-Z])/g, "$1 $2").split(" ");
  return words.map((word, index) => (index === 0 ? word : word.toLowerCase())).join(" ");
}

function sentenceCase(text: string): string {
  const spaced = text.replace(/([a-z0-9])([A-Z])/g, "$1 $2").toLowerCase();
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
}

export function humanizeIntelFieldPath(fieldPath: string): string {
  const parts = fieldPath.split(".").filter((part) => part !== "");
  if (parts.length === 0) {
    return INTEL_EMPTY_VALUE;
  }
  if (parts[0] === "basics" && parts.length >= 3) {
    return sentenceCase(`${parts[1]} ${parts[2]}`);
  }
  return sentenceCase(parts[parts.length - 1]);
}

function formatIntelAge(days: number): string {
  const age = ageFromDays(days);
  if (!age) {
    return INTEL_EMPTY_VALUE;
  }
  return new Intl.NumberFormat(intlLocale(), {
    style: "unit",
    unit: age.unit,
    unitDisplay: "short",
    maximumFractionDigits: 0,
  }).format(age.value);
}

/**
 * Renders a carrier intelligence value the way a person reads it. Event values
 * arrive as strings (a JSON encoding for anything that is not already text), so
 * the field path decides the shape: epoch seconds on `...At`/`...Date` become a
 * date, insurance amounts become currency, ages become years, booleans become
 * Yes/No and a missing value becomes an em dash.
 */
export function formatIntelValue(
  fieldPath: string | null | undefined,
  value: string | number | boolean | null | undefined,
  labels: IntelValueLabels = DEFAULT_INTEL_VALUE_LABELS,
): string {
  if (value === null || value === undefined) {
    return labels.empty;
  }
  if (typeof value === "boolean") {
    return value ? labels.yes : labels.no;
  }

  const text = unquoteIntelValue(String(value).trim());
  if (text === "" || text === "null") {
    return labels.empty;
  }
  if (text === "true") {
    return labels.yes;
  }
  if (text === "false") {
    return labels.no;
  }

  const path = fieldPath ?? "";
  const leaf = path.slice(path.lastIndexOf(".") + 1);

  if (INTEL_MONEY_PATH.test(path) && isDecimalString(text)) {
    return formatCurrency(Number(text));
  }
  if (INTEL_DATE_LEAF.test(leaf) && INTEL_INTEGER.test(text)) {
    const seconds = Number(text);
    return seconds > 0 ? formatUnixDateMedium(seconds, { timezone: "UTC" }) : labels.empty;
  }
  if (INTEL_AGE_DAYS_LEAF.test(leaf) && INTEL_INTEGER.test(text)) {
    return formatIntelAge(Number(text));
  }
  if (INTEL_ENUM_PATH.test(path)) {
    return humanizeIntelEnum(text);
  }
  return text;
}
