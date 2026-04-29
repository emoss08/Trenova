import { z } from "zod";
import { optionalStringSchema, timestampSchema, versionSchema } from "./helpers";

export const sequenceTypes = [
  "pro_number",
  "consolidation",
  "invoice",
  "work_order",
  "journal_batch",
  "journal_entry",
  "manual_journal_request",
  "location_code",
] as const;

export const sequenceTypeSchema = z.enum(sequenceTypes);

export const locationCodeComponents = ["name", "city", "state", "postal_code"] as const;

export const locationCodeComponentSchema = z.enum(locationCodeComponents);

const defaultLocationCodeStrategy = {
  components: ["name", "city", "state"],
  componentWidth: 3,
  sequenceDigits: 3,
  separator: "-",
  casing: "upper",
  fallbackPrefix: "LOC",
} as const;

function normalizeLocationCodeStrategy(value: unknown) {
  if (!value || typeof value !== "object") {
    return defaultLocationCodeStrategy;
  }

  const input = value as Record<string, unknown>;
  const components = Array.isArray(input.components)
    ? input.components
    : defaultLocationCodeStrategy.components;
  const componentWidth =
    typeof input.componentWidth === "number" && input.componentWidth > 0
      ? input.componentWidth
      : typeof input.prefixLength === "number" && input.prefixLength > 0
        ? input.prefixLength
        : defaultLocationCodeStrategy.componentWidth;
  const sequenceDigits =
    typeof input.sequenceDigits === "number" && input.sequenceDigits > 0
      ? input.sequenceDigits
      : defaultLocationCodeStrategy.sequenceDigits;
  const separator = typeof input.separator === "string" ? input.separator : defaultLocationCodeStrategy.separator;
  const casing = input.casing === "lower" ? "lower" : defaultLocationCodeStrategy.casing;
  const fallbackPrefix =
    typeof input.fallbackPrefix === "string" && input.fallbackPrefix.trim()
      ? input.fallbackPrefix
      : defaultLocationCodeStrategy.fallbackPrefix;

  return {
    components,
    componentWidth,
    sequenceDigits,
    separator,
    casing,
    fallbackPrefix,
  };
}

export const locationCodeStrategySchema = z
  .preprocess(normalizeLocationCodeStrategy, z
  .object({
    components: z.array(locationCodeComponentSchema).min(1),
    componentWidth: z.number().int().min(1).max(10),
    sequenceDigits: z.number().int().min(1).max(10),
    separator: z.string().max(2),
    casing: z.enum(["upper", "lower"]),
    fallbackPrefix: z.string().min(1).max(32),
  }))
  .refine((data) => !data.separator || ["-", "_", "/", "."].includes(data.separator), {
    path: ["separator"],
    message: "Separator must be one of '-', '_', '/', '.', or blank",
  })
  .refine(
    (data) =>
      data.components.length * data.componentWidth +
        data.sequenceDigits +
        Math.max(data.components.length, 0) * (data.separator ? data.separator.length : 0) <=
      32,
    {
      path: ["sequenceDigits"],
      message: "Components, separators, and sequence digits must fit within 32 characters",
    },
  );

export const sequenceConfigSchema = z
  .object({
    id: optionalStringSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    sequenceType: sequenceTypeSchema,
    prefix: z.string().min(1, "Prefix is required").max(20),
    includeYear: z.boolean(),
    yearDigits: z.number().int().min(2).max(4),
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
  .refine(
    (data) =>
      !data.includeRandomDigits || (data.randomDigitsCount >= 1 && data.randomDigitsCount <= 10),
    {
      path: ["randomDigitsCount"],
      message: "Random digits count must be between 1 and 10 when include random digits is enabled",
    },
  )
  .refine((data) => !data.useSeparators || ["-", "_", "/", "."].includes(data.separatorChar), {
    path: ["separatorChar"],
    message: "Separator must be one of '-', '_', '/', '.'",
  })
  .refine((data) => !data.allowCustomFormat || data.customFormat.trim().length > 0, {
    path: ["customFormat"],
    message: "Custom format is required when custom format is enabled",
  });

export const sequenceConfigDocumentSchema = z.object({
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  configs: z
    .array(sequenceConfigSchema)
    .length(
      sequenceTypes.length,
      `Exactly ${sequenceTypes.length} sequence configurations are required`,
    ),
});

export type SequenceType = z.infer<typeof sequenceTypeSchema>;
export type LocationCodeComponent = z.infer<typeof locationCodeComponentSchema>;
export type LocationCodeStrategy = z.infer<typeof locationCodeStrategySchema>;
export type SequenceConfig = z.infer<typeof sequenceConfigSchema>;
export type SequenceConfigDocument = z.infer<typeof sequenceConfigDocumentSchema>;
