import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const sequenceTypes = [
  "pro_number",
  "consolidation",
  "order",
  "invoice",
  "work_order",
  "journal_batch",
  "journal_entry",
  "manual_journal_request",
  "location_code",
  "driver_settlement",
  "carrier_settlement",
  "invoice_run",
  "factoring_submission",
] as const;

export const sequenceTypeSchema = z.enum(sequenceTypes);

export const locationCodeComponents = ["name", "city", "state", "postal_code"] as const;

export const locationCodeComponentSchema = z.enum(locationCodeComponents);

export const sequenceSeparators = ["-", "_", "/", "."] as const;

export const yearDigitChoices = [2, 4] as const;

/** The longest location code the server issues, separators and counter included. */
export const MAX_LOCATION_CODE_LENGTH = 32;

const defaultLocationCodeStrategy = {
  components: ["name", "city", "state"],
  componentWidth: 3,
  sequenceDigits: 3,
  separator: "-",
  casing: "upper",
  fallbackPrefix: "LOC",
} as const;

/** What the server keeps of a name: its letters and digits, as `NormalizeIdentifier` does. */
export function identifierCharacters(value: string): string {
  return value.replace(/[^\p{L}\p{Nd}]/gu, "");
}

/** The length of the longest code a strategy can issue, as the server measures it. */
export function locationCodeLength(strategy: {
  components: readonly unknown[];
  componentWidth: number;
  sequenceDigits: number;
  separator: string;
}): number {
  return (
    strategy.components.length * strategy.componentWidth +
    strategy.sequenceDigits +
    strategy.components.length * strategy.separator.length
  );
}

export const locationCodeStrategySchema = z
  .preprocess(
    (value) => value ?? defaultLocationCodeStrategy,
    z.object({
      components: z.array(locationCodeComponentSchema).min(1, {
        error: () => translate("Choose at least one part to build the code from"),
      }),
      componentWidth: z.number().int().min(1).max(10),
      sequenceDigits: z.number().int().min(1).max(10),
      separator: z.string().max(2),
      casing: z.enum(["upper", "lower"]),
      fallbackPrefix: z.string().max(MAX_LOCATION_CODE_LENGTH),
    }),
  )
  .superRefine((data, ctx) => {
    if (data.separator && !(sequenceSeparators as readonly string[]).includes(data.separator)) {
      ctx.addIssue({
        code: "custom",
        path: ["separator"],
        message: translate("Choose a hyphen, underscore, slash, period or none"),
      });
    }
    if (identifierCharacters(data.fallbackPrefix.trim()) === "") {
      ctx.addIssue({
        code: "custom",
        path: ["fallbackPrefix"],
        message: translate("Enter a fallback with at least one letter or digit"),
      });
    }
    if (locationCodeLength(data) > MAX_LOCATION_CODE_LENGTH) {
      ctx.addIssue({
        code: "custom",
        path: ["sequenceDigits"],
        message: translate(
          "A code can be at most {0} characters with its separators and counter. Use fewer parts or shorter ones.",
          MAX_LOCATION_CODE_LENGTH,
        ),
      });
    }
  });

export const sequenceConfigSchema = z
  .object({
    id: optionalStringSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    sequenceType: sequenceTypeSchema,
    prefix: z.string().max(20),
    includeYear: z.boolean(),
    yearDigits: z.number().int().min(0).max(4),
    includeMonth: z.boolean(),
    includeWeekNumber: z.boolean(),
    includeDay: z.boolean(),
    sequenceDigits: z.number().int().min(1).max(10),
    includeLocationCode: z.boolean(),
    includeRandomDigits: z.boolean(),
    randomDigitsCount: z.number().int().min(0).max(10),
    includeCheckDigit: z.boolean(),
    includeBusinessUnitCode: z.boolean(),
    useSeparators: z.boolean(),
    separatorChar: z.string().max(2),
    allowCustomFormat: z.boolean(),
    customFormat: z.string(),
    locationCodeStrategy: locationCodeStrategySchema.nullish(),
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
  })
  .superRefine((data, ctx) => {
    if (data.prefix.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["prefix"],
        message: translate("Enter a prefix"),
      });
    }
    if (
      data.includeYear &&
      !(yearDigitChoices as readonly number[]).includes(data.yearDigits)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["yearDigits"],
        message: translate("Choose a two- or four-digit year"),
      });
    }
    if (data.includeRandomDigits && (data.randomDigitsCount < 1 || data.randomDigitsCount > 10)) {
      ctx.addIssue({
        code: "custom",
        path: ["randomDigitsCount"],
        message: translate("Add between 1 and 10 random digits"),
      });
    }
    if (
      data.useSeparators &&
      !(sequenceSeparators as readonly string[]).includes(data.separatorChar)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["separatorChar"],
        message: translate("Choose a hyphen, underscore, slash or period"),
      });
    }
    if (data.allowCustomFormat && data.customFormat.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["customFormat"],
        message: translate("Write the pattern, or turn off your own pattern"),
      });
    }
  });

export const sequenceConfigDocumentSchema = z.object({
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  configs: z.array(sequenceConfigSchema).length(sequenceTypes.length, {
    error: () =>
      translate("Exactly {0} sequence configurations are required", sequenceTypes.length),
  }),
});

export type SequenceType = z.infer<typeof sequenceTypeSchema>;
export type LocationCodeComponent = z.infer<typeof locationCodeComponentSchema>;
export type LocationCodeStrategy = z.infer<typeof locationCodeStrategySchema>;
export type SequenceConfig = z.infer<typeof sequenceConfigSchema>;
export type SequenceConfigDocument = z.infer<typeof sequenceConfigDocumentSchema>;
