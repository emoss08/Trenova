import type {
  CarrierIntelDepth,
  CarrierIntelLookupInput,
  CarrierSourcingSearchInput,
} from "@trenova/graphql/generated/graphql";
import { z } from "zod";

export const SOURCING_PAGE_SIZE = 25;
export const SOURCING_TEXT_MAX_LENGTH = 200;
export const CARRIER_CODE_MAX_LENGTH = 10;

const optionalCount = z
  .number({ error: "Enter a whole number" })
  .int("Enter a whole number")
  .min(0, "Cannot be negative")
  .nullable();

export const sourcingSearchSchema = z
  .object({
    text: z
      .string()
      .trim()
      .max(SOURCING_TEXT_MAX_LENGTH, "Search text cannot exceed 200 characters"),
    state: z.string(),
    originState: z.string(),
    destinationState: z.string(),
    minPowerUnits: optionalCount,
    maxPowerUnits: optionalCount,
    minAuthorityAgeDays: optionalCount,
    hazmatOnly: z.boolean(),
    excludeBlocking: z.boolean(),
    excludeExistingCarriers: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (
      values.text === "" &&
      values.state === "" &&
      values.originState === "" &&
      values.destinationState === ""
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["text"],
        message: "Enter a name, EIN or VIN, or choose a state to search",
      });
    }
    if (
      values.minPowerUnits !== null &&
      values.maxPowerUnits !== null &&
      values.maxPowerUnits < values.minPowerUnits
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["maxPowerUnits"],
        message: "Maximum power units must be at least the minimum",
      });
    }
  });

export type SourcingSearchFormValues = z.infer<typeof sourcingSearchSchema>;

export const SOURCING_SEARCH_DEFAULTS: SourcingSearchFormValues = {
  text: "",
  state: "",
  originState: "",
  destinationState: "",
  minPowerUnits: null,
  maxPowerUnits: null,
  minAuthorityAgeDays: null,
  hazmatOnly: false,
  excludeBlocking: false,
  excludeExistingCarriers: false,
};

function blankToNull(value: string): string | null {
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}

export function toSourcingSearchInput(
  values: SourcingSearchFormValues,
  offset: number,
  limit: number = SOURCING_PAGE_SIZE,
): CarrierSourcingSearchInput {
  return {
    text: blankToNull(values.text),
    state: blankToNull(values.state),
    originState: blankToNull(values.originState),
    destinationState: blankToNull(values.destinationState),
    minPowerUnits: values.minPowerUnits,
    maxPowerUnits: values.maxPowerUnits,
    minAuthorityAgeDays: values.minAuthorityAgeDays,
    hazmatOnly: values.hazmatOnly,
    excludeBlocking: values.excludeBlocking,
    excludeExistingCarriers: values.excludeExistingCarriers,
    limit,
    offset: Math.max(offset, 0),
  };
}

export const LOOKUP_KINDS = ["dot", "mc"] as const;
export const LOOKUP_DEPTHS = [
  "Full",
  "Lite",
  "FMCSA",
] as const satisfies readonly CarrierIntelDepth[];
export type LookupKind = (typeof LOOKUP_KINDS)[number];

export const prospectLookupSchema = z.object({
  kind: z.enum(LOOKUP_KINDS),
  number: z
    .string()
    .trim()
    .min(1, "Enter a number to look up")
    .regex(/^(MC-?)?\d{1,8}$/i, "Enter digits only, up to 8 of them"),
  depth: z.enum(LOOKUP_DEPTHS).nullable(),
});

export type ProspectLookupFormValues = z.infer<typeof prospectLookupSchema>;

export function toLookupInput(values: ProspectLookupFormValues): CarrierIntelLookupInput {
  const digits = values.number.replace(/\D/g, "");
  return values.kind === "dot"
    ? { dotNumber: digits, depth: values.depth }
    : { docketNumber: digits, depth: values.depth };
}

export const importSourcedCarrierSchema = z.object({
  code: z.string().trim().max(CARRIER_CODE_MAX_LENGTH, "Code cannot exceed 10 characters"),
  enrollMonitoring: z.boolean(),
});

export type ImportSourcedCarrierFormValues = z.infer<typeof importSourcedCarrierSchema>;
