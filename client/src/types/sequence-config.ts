import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const sequenceTypes = [
  "pro_number",
  "consolidation",
  "invoice",
  "work_order",
  "journal_batch",
  "journal_entry",
  "manual_journal_request",
] as const;

export const sequenceTypeSchema = z.enum(sequenceTypes);

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
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
  })
  .refine(
    (data) =>
      !data.includeRandomDigits ||
      (data.randomDigitsCount >= 1 && data.randomDigitsCount <= 10),
    {
      path: ["randomDigitsCount"],
      message:
        "Random digits count must be between 1 and 10 when include random digits is enabled",
    },
  )
  .refine(
    (data) =>
      !data.useSeparators || ["-", "_", "/", "."].includes(data.separatorChar),
    {
      path: ["separatorChar"],
      message: "Separator must be one of '-', '_', '/', '.'",
    },
  )
  .refine(
    (data) => !data.allowCustomFormat || data.customFormat.trim().length > 0,
    {
      path: ["customFormat"],
      message: "Custom format is required when custom format is enabled",
    },
  );

export const sequenceConfigDocumentSchema = z.object({
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  configs: z
    .array(sequenceConfigSchema)
    .length(sequenceTypes.length, `Exactly ${sequenceTypes.length} sequence configurations are required`),
});

export type SequenceType = z.infer<typeof sequenceTypeSchema>;
export type SequenceConfig = z.infer<typeof sequenceConfigSchema>;
export type SequenceConfigDocument = z.infer<typeof sequenceConfigDocumentSchema>;
