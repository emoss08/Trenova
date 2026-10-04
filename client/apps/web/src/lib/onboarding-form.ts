import { timezoneGroupedChoices } from "@/lib/choices";
import type { OnboardingFormValues, OnboardingState, OperationType } from "@/types/onboarding";
import type { FieldPath } from "react-hook-form";

export type OnboardingStepId = "company" | "operation" | "sample-data" | "review";

export type OnboardingStepDefinition = {
  id: OnboardingStepId;
  label: string;
  detail: string;
  fields: readonly FieldPath<OnboardingFormValues>[];
};

export const ONBOARDING_STEPS: readonly OnboardingStepDefinition[] = [
  {
    id: "company",
    label: "Company profile",
    detail: "Name, timezone and address",
    fields: [
      "organization.name",
      "organization.timezone",
      "organization.addressLine1",
      "organization.city",
      "organization.stateId",
      "organization.postalCode",
      "organization.scacCode",
      "organization.dotNumber",
    ],
  },
  {
    id: "operation",
    label: "Operation type",
    detail: "Trucks, brokerage or both",
    fields: ["operationType"],
  },
  {
    id: "sample-data",
    label: "Sample data",
    detail: "Start empty or with examples",
    fields: ["loadSampleData"],
  },
  {
    id: "review",
    label: "Review",
    detail: "Confirm and finish",
    fields: [],
  },
];

export type OperationTypeDefinition = {
  value: OperationType;
  label: string;
  description: string;
};

export const OPERATION_TYPES: readonly OperationTypeDefinition[] = [
  {
    value: "asset",
    label: "Asset carrier",
    description:
      "You haul freight with your own trucks and drivers. Dispatch, fleet, hours of service and driver pay lead the product.",
  },
  {
    value: "brokerage",
    label: "Freight brokerage",
    description:
      "You arrange freight with outside carriers. Carrier sourcing, tenders, rate confirmations and carrier settlements lead the product.",
  },
  {
    value: "both",
    label: "Both",
    description:
      "You run your own fleet and broker loads to partner carriers. Every part of the product is turned on.",
  },
];

export function operationTypeLabel(value: OperationType | undefined): string {
  return OPERATION_TYPES.find((type) => type.value === value)?.label ?? "";
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
