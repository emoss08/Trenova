import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const OnboardingStatus = z.enum(["pending", "completed"]);
export type OnboardingStatus = z.infer<typeof OnboardingStatus>;

export const OperationType = z.enum(["asset", "brokerage", "both"]);
export type OperationType = z.infer<typeof OperationType>;

const optionalText = z
  .string()
  .nullish()
  .transform((value) => value ?? "");

export const onboardingOrganizationSchema = z
  .object({
    name: optionalText,
    timezone: optionalText,
    addressLine1: optionalText,
    city: optionalText,
    stateId: optionalText,
    postalCode: optionalText,
    scacCode: optionalText,
    dotNumber: optionalText,
  })
  .partial();

export const onboardingStateSchema = z.object({
  required: z.boolean().catch(false).default(false),
  status: OnboardingStatus.catch("completed").default("completed"),
  operationType: OperationType.nullish().catch(null),
  sampleDataLoaded: z.boolean().catch(false).default(false),
  completedAt: z.number().nullish().catch(null),
  organization: onboardingOrganizationSchema.nullish().catch(null),
});

export type OnboardingState = z.infer<typeof onboardingStateSchema>;

export const ONBOARDING_NOT_REQUIRED: OnboardingState = {
  required: false,
  status: "completed",
  operationType: null,
  sampleDataLoaded: false,
  completedAt: null,
  organization: null,
};

const SCAC_PATTERN = /^[A-Z]{2,4}$/;
const DOT_PATTERN = /^\d{1,8}$/;

export const onboardingFormSchema = z.object({
  organization: z.object({
    name: z
      .string()
      .trim()
      .min(1, { error: () => translate("Enter your company name") })
      .max(100, { error: () => translate("Company name must be 100 characters or fewer") }),
    timezone: z.string().min(1, { error: () => translate("Select a timezone") }),
    addressLine1: z
      .string()
      .trim()
      .min(1, { error: () => translate("Enter a street address") })
      .max(150, { error: () => translate("Address must be 150 characters or fewer") }),
    city: z
      .string()
      .trim()
      .min(1, { error: () => translate("Enter a city") })
      .max(100, { error: () => translate("City must be 100 characters or fewer") }),
    stateId: z.string().min(1, { error: () => translate("Select a state") }),
    postalCode: z
      .string()
      .trim()
      .regex(/^\d{5}(-\d{4})?$/, { error: () => translate("Enter a 5-digit ZIP code") }),
    scacCode: z
      .string()
      .trim()
      .transform((value) => value.toUpperCase())
      .refine((value) => value === "" || SCAC_PATTERN.test(value), {
        error: () => translate("A SCAC is 2 to 4 letters"),
      }),
    dotNumber: z
      .string()
      .trim()
      .refine((value) => value === "" || DOT_PATTERN.test(value), {
        error: () => translate("A DOT number is up to 8 digits"),
      }),
  }),
  operationType: OperationType,
  loadSampleData: z.boolean(),
});

export type OnboardingFormValues = z.input<typeof onboardingFormSchema>;
export type OnboardingFormOutput = z.output<typeof onboardingFormSchema>;

export type CompleteOnboardingRequest = {
  organization: {
    name: string;
    timezone: string;
    addressLine1: string;
    city: string;
    stateId: string;
    postalCode: string;
    scacCode?: string;
    dotNumber?: string;
  };
  operationType: OperationType;
  loadSampleData: boolean;
};

export function toCompleteOnboardingRequest(
  values: OnboardingFormOutput,
): CompleteOnboardingRequest {
  const { scacCode, dotNumber, ...organization } = values.organization;
  return {
    organization: {
      ...organization,
      ...(scacCode === "" ? {} : { scacCode }),
      ...(dotNumber === "" ? {} : { dotNumber }),
    },
    operationType: values.operationType,
    loadSampleData: values.loadSampleData,
  };
}
