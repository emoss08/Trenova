import { z } from "zod";
import { decimalStringSchema, optionalStringSchema, tenantInfoSchema } from "@trenova/shared/types/helpers";

export const rateTableLookupTypeSchema = z.enum(["Exact", "Range"]);

export type RateTableLookupType = z.infer<typeof rateTableLookupTypeSchema>;

const entryValueSchema = z.preprocess(
  (val) => {
    if (val === "" || val === null || val === undefined) return undefined;
    const parsed = parseFloat(
      typeof val === "string" || typeof val === "number" ? String(val) : "",
    );
    return isNaN(parsed) ? undefined : parsed;
  },
  z.number({ message: "Value is required" }),
);

export const rateTableEntrySchema = z.object({
  id: optionalStringSchema,
  rateTableId: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  matchKey: z
    .string()
    .max(100, { message: "Match key must be less than 100 characters" })
    .nullish(),
  rangeMin: decimalStringSchema,
  rangeMax: decimalStringSchema,
  value: entryValueSchema,
  sortOrder: z.number().int().min(0).default(0),
  createdAt: z.number().int().positive().optional(),
  updatedAt: z.number().int().positive().optional(),
});

export type RateTableEntry = z.infer<typeof rateTableEntrySchema>;

export const rateTableRowSchema = z.object({
  ...tenantInfoSchema.shape,
  name: z
    .string()
    .min(1, { message: "Name is required" })
    .max(100, { message: "Name must be less than 100 characters" }),
  key: z
    .string()
    .min(1, { message: "Key is required" })
    .max(64, { message: "Key must be less than 64 characters" })
    .regex(/^[a-zA-Z][a-zA-Z0-9_]*$/, {
      message: "Key must start with a letter and contain only letters, numbers, and underscores",
    }),
  description: optionalStringSchema,
  lookupType: rateTableLookupTypeSchema,
  active: z.boolean().default(true),
});

export type RateTableRow = z.infer<typeof rateTableRowSchema>;

export const rateTableSchema = rateTableRowSchema
  .extend({
    entries: z.array(rateTableEntrySchema).min(1, { message: "At least one entry is required" }),
  })
  .superRefine((data, ctx) => {
    if (data.lookupType === "Exact") {
      const seen = new Set<string>();
      data.entries.forEach((entry, index) => {
        const matchKey = entry.matchKey?.trim() ?? "";
        if (!matchKey) {
          ctx.addIssue({
            code: "custom",
            path: ["entries", index, "matchKey"],
            message: "Match key is required for exact lookup entries",
          });
          return;
        }
        if (seen.has(matchKey)) {
          ctx.addIssue({
            code: "custom",
            path: ["entries", index, "matchKey"],
            message: "Match key must be unique",
          });
          return;
        }
        seen.add(matchKey);
      });
      return;
    }

    data.entries.forEach((entry, index) => {
      if (entry.rangeMin === null || entry.rangeMin === undefined) {
        ctx.addIssue({
          code: "custom",
          path: ["entries", index, "rangeMin"],
          message: "Range min is required for range lookup entries",
        });
        return;
      }
      if (
        entry.rangeMax !== null &&
        entry.rangeMax !== undefined &&
        entry.rangeMax <= entry.rangeMin
      ) {
        ctx.addIssue({
          code: "custom",
          path: ["entries", index, "rangeMax"],
          message: "Range max must be greater than range min",
        });
      }
    });
  })
  .transform((data) => ({
    ...data,
    entries: data.entries.map((entry, index) => ({
      ...entry,
      sortOrder: index,
      matchKey: data.lookupType === "Exact" ? (entry.matchKey?.trim() ?? "") : null,
      rangeMin: data.lookupType === "Range" ? entry.rangeMin : null,
      rangeMax: data.lookupType === "Range" ? entry.rangeMax : null,
    })),
  }));

export type RateTable = z.infer<typeof rateTableSchema>;
export type RateTableFormValues = RateTable;
