import { timezoneGroupedChoices } from "@/lib/choices";
import type { OnboardingFormValues, OnboardingState, OperationType } from "@/types/onboarding";
import type { FieldPath } from "react-hook-form";

export type OnboardingStepId =
  | "name"
  | "timezone"
  | "address"
  | "ids"
  | "operation"
  | "sample-data"
  | "review";

export type OnboardingStepDefinition = {
  id: OnboardingStepId;
  fields: readonly FieldPath<OnboardingFormValues>[];
};

/** The conversation's turns, in order: Nova asks one thing per turn. */
export const ONBOARDING_STEPS: readonly OnboardingStepDefinition[] = [
  { id: "name", fields: ["organization.name"] },
  { id: "timezone", fields: ["organization.timezone"] },
  {
    id: "address",
    fields: [
      "organization.addressLine1",
      "organization.city",
      "organization.stateId",
      "organization.postalCode",
    ],
  },
  { id: "ids", fields: ["organization.scacCode", "organization.dotNumber"] },
  { id: "operation", fields: ["operationType"] },
  { id: "sample-data", fields: ["loadSampleData"] },
  { id: "review", fields: [] },
];

export const REVIEW_STEP_INDEX = ONBOARDING_STEPS.length - 1;

export function onboardingStepIndex(id: OnboardingStepId): number {
  return ONBOARDING_STEPS.findIndex((step) => step.id === id);
}

/**
 * Where an answer leads: the next turn, unless the person had already been further
 * and came back to edit, in which case straight back to the furthest turn reached.
 */
export function nextOnboardingStep(current: number, furthest: number): number {
  return Math.min(furthest > current ? furthest : current + 1, REVIEW_STEP_INDEX);
}

/** The top progress bar: 4% at the first turn, 78% at the review, 92% building, 100% ready. */
export function onboardingProgress(
  current: number,
  phase: "setup" | "building" | "ready" | "failed",
): number {
  if (phase === "ready") {
    return 100;
  }
  if (phase !== "setup") {
    return 92;
  }
  return Math.round((current / ONBOARDING_STEPS.length) * 86) + 4;
}

/** The first turn holding a field the predicate reports invalid, or -1. */
export function firstInvalidOnboardingStep(
  isInvalid: (field: FieldPath<OnboardingFormValues>) => boolean,
): number {
  return ONBOARDING_STEPS.findIndex((step) => step.fields.some(isInvalid));
}

export type OperationTypeDefinition = {
  value: OperationType;
  label: string;
  description: string;
  modules: readonly string[];
};

const ASSET_MODULES = ["Dispatch", "Fleet", "Hours of service", "Driver pay"] as const;
const BROKERAGE_MODULES = [
  "Carrier sourcing",
  "Tenders",
  "Rate confirmations",
  "Carrier settlements",
] as const;

export const OPERATION_TYPES: readonly OperationTypeDefinition[] = [
  {
    value: "asset",
    label: "Asset carrier",
    description:
      "You haul freight with your own trucks and drivers. Dispatch, fleet, hours of service and driver pay lead the product.",
    modules: ASSET_MODULES,
  },
  {
    value: "brokerage",
    label: "Freight brokerage",
    description:
      "You arrange freight with outside carriers. Carrier sourcing, tenders, rate confirmations and carrier settlements lead the product.",
    modules: BROKERAGE_MODULES,
  },
  {
    value: "both",
    label: "Both",
    description:
      "You run your own fleet and broker loads to partner carriers. Every part of the product is turned on.",
    modules: [...ASSET_MODULES, ...BROKERAGE_MODULES],
  },
];

export function operationTypeDefinition(
  value: OperationType | undefined,
): OperationTypeDefinition | undefined {
  return OPERATION_TYPES.find((type) => type.value === value);
}

/**
 * What "Load sample data" creates, as the server's onboarding service does. Each row
 * names the meter it counts against, so the person can see how much of the demo it
 * spends.
 */
export const SAMPLE_DATA_SET: readonly { meter: string; quantity: number }[] = [
  { meter: "customers.total", quantity: 2 },
  { meter: "locations.total", quantity: 4 },
  { meter: "workers.total", quantity: 1 },
  { meter: "tractors.total", quantity: 1 },
  { meter: "trailers.total", quantity: 1 },
  { meter: "shipments.total", quantity: 2 },
];

export const SAMPLE_DATA_RECORD_COUNT = SAMPLE_DATA_SET.reduce(
  (total, row) => total + row.quantity,
  0,
);

/**
 * The zones the timezone turn offers as tiles, in the order Nova lists them, with the
 * short names Nova uses for them. Every other zone keeps its picker label.
 */
const US_TIMEZONES: readonly { zone: string; label: string }[] = [
  { zone: "America/New_York", label: "Eastern time" },
  { zone: "America/Chicago", label: "Central time" },
  { zone: "America/Denver", label: "Mountain time" },
  { zone: "America/Phoenix", label: "Arizona" },
  { zone: "America/Los_Angeles", label: "Pacific time" },
  { zone: "America/Anchorage", label: "Alaska time" },
  { zone: "Pacific/Honolulu", label: "Hawaii time" },
];

/** The name the conversation gives a zone (untranslated; callers pass it through `t`). */
export function onboardingTimezoneLabel(zone: string): string {
  return US_TIMEZONES.find((entry) => entry.zone === zone)?.label ?? timezoneLabel(zone);
}

/**
 * The timezone tiles: the browser's zone first when the picker knows it (even one
 * outside the US), then the US zones in their usual order.
 */
export function onboardingTimezoneTiles(detected: string): string[] {
  const tiles: string[] = KNOWN_TIMEZONES.has(detected) ? [detected] : [];
  for (const { zone } of US_TIMEZONES) {
    if (zone !== detected) {
      tiles.push(zone);
    }
  }
  return tiles;
}

export function isKnownTimezone(zone: string): boolean {
  return KNOWN_TIMEZONES.has(zone);
}

const KNOWN_TIMEZONES = new Set(
  timezoneGroupedChoices.flatMap((group) => group.options.map((option) => String(option.value))),
);

export function timezoneLabel(timezone: string): string {
  for (const group of timezoneGroupedChoices) {
    const match = group.options.find((option) => option.value === timezone);
    if (match) {
      return match.label;
    }
  }
  return timezone;
}

/** The browser's zone when the timezone picker offers it, so most people need not choose. */
export function browserTimezone(): string {
  try {
    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    return KNOWN_TIMEZONES.has(zone) ? zone : "";
  } catch {
    return "";
  }
}

/**
 * The wizard's starting values: whatever the organization already holds (the server
 * prefills what signup collected), then the name the session knows, then the
 * browser's timezone. A brand-new demo has nothing loaded, so sample data is offered
 * on by default; operation type starts on "both" so nothing is hidden by accident.
 */
export function onboardingFormDefaults({
  state,
  organizationName,
  fallbackTimezone,
}: {
  state: OnboardingState | null | undefined;
  organizationName: string | undefined;
  fallbackTimezone: string;
}): OnboardingFormValues {
  const organization = state?.organization ?? {};
  const timezone = organization.timezone ?? "";

  return {
    organization: {
      name: organization.name || organizationName || "",
      timezone: KNOWN_TIMEZONES.has(timezone) ? timezone : fallbackTimezone,
      addressLine1: organization.addressLine1 ?? "",
      city: organization.city ?? "",
      stateId: organization.stateId ?? "",
      postalCode: organization.postalCode ?? "",
      scacCode: organization.scacCode ?? "",
      dotNumber: organization.dotNumber ?? "",
    },
    operationType: state?.operationType ?? "both",
    loadSampleData: state?.sampleDataLoaded ? false : true,
  };
}
