// formula-library.ts types formula libraries: an organisation's Tariff `library` files, imported
// by tariffs and other libraries as `use org.<name>` (docs/engineering/formula-language.md,
// "Modules and the standard library"). A library is versioned and reviewed like a template; a
// template version pins the library versions it imports, so publishing a library version never
// reprices a contract.

import { translate } from "@trenova/shared/i18n/runtime";
import { linkedRecordSchema } from "@trenova/shared/lib/linked-record";
import { z } from "zod";
import {
  fieldChangeSchema,
  formulaReviewDecisionSchema,
  formulaTemplateStatusSchema,
  versionTagSchema,
} from "./formula-template";
import { userSchema } from "./user";

/** A library's name is the module path segment after `org.`: a camelCase name. */
export const FORMULA_LIBRARY_NAME_PATTERN = /^[a-z][A-Za-z0-9]*$/;
export const FORMULA_LIBRARY_NAME_MAX = 64;

/** Libraries move through the same review states as templates. */
export const formulaLibraryStatusSchema = formulaTemplateStatusSchema;
export type FormulaLibraryStatus = z.infer<typeof formulaLibraryStatusSchema>;

export const formulaLibrarySchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") })
    .max(FORMULA_LIBRARY_NAME_MAX, {
      error: () => translate("Name must be {0} characters or fewer", FORMULA_LIBRARY_NAME_MAX),
    })
    .regex(FORMULA_LIBRARY_NAME_PATTERN, {
      error: () =>
        translate(
          "A library name starts with a lowercase letter and holds only letters and digits, like accessorials",
        ),
    }),
  description: z.string().default(""),
  source: z.string().refine((value) => value.trim().length > 0, {
    error: () => translate("The library's source is required"),
  }),
  status: formulaLibraryStatusSchema.default("Draft"),
  testCount: z.number().int().default(0),
  submittedById: z.string().nullish(),
  submittedAt: z.number().nullish(),
  approvedById: z.string().nullish(),
  approvedAt: z.number().nullish(),
  reviewComment: z.string().nullish(),
  version: z.number().optional(),
  currentVersionNumber: z.number().optional(),
  /** The latest approved version, the one a new template version would pin. */
  approvedVersionNumber: z.number().nullish(),
  /** How many template versions import this library. */
  usageCount: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});
export type FormulaLibrary = z.output<typeof formulaLibrarySchema>;
export type FormulaLibraryFormValues = z.input<typeof formulaLibrarySchema>;

/** What a save writes. */
export type FormulaLibraryWrite = Pick<FormulaLibrary, "name" | "description" | "source"> & {
  version?: number;
};

export const formulaLibraryVersionSchema = z.object({
  id: z.string(),
  libraryId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  versionNumber: z.number(),
  name: z.string(),
  description: z.string().optional(),
  source: z.string(),
  status: formulaLibraryStatusSchema,
  testCount: z.number().int().default(0),
  changeMessage: z.string().optional(),
  changeSummary: z.record(z.string(), fieldChangeSchema).nullish(),
  tags: z
    .array(versionTagSchema)
    .nullish()
    .transform((v) => v ?? []),
  createdById: z.string(),
  createdAt: z.number(),
  createdBy: linkedRecordSchema("user", userSchema).nullish(),
});
export type FormulaLibraryVersion = z.output<typeof formulaLibraryVersionSchema>;

export const formulaLibraryReviewSchema = z.object({
  id: z.string(),
  libraryId: z.string(),
  round: z.number().int(),
  decision: formulaReviewDecisionSchema,
  actorId: z.string().nullish(),
  comment: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
  baseVersionNumber: z.number().int().default(0),
  createdAt: z.number(),
  actor: linkedRecordSchema("user", userSchema).nullish(),
  requestedReviewerId: z.string().nullish(),
  requestedReviewer: linkedRecordSchema("user", userSchema).nullish(),
  reviewerNote: z
    .string()
    .nullish()
    .transform((v) => v ?? ""),
});
export type FormulaLibraryReview = z.output<typeof formulaLibraryReviewSchema>;

/**
 * A template version that imports the library, and the library version it pinned, for "who
 * uses this".
 */
export const formulaLibraryUsageSchema = z.object({
  templateId: z.string(),
  templateName: z.string(),
  templateStatus: formulaTemplateStatusSchema,
  templateVersion: z.number().int(),
  versionStatus: formulaTemplateStatusSchema,
  libraryVersionNumber: z.number().int(),
});
export type FormulaLibraryUsage = z.output<typeof formulaLibraryUsageSchema>;

export const formulaLibraryUsagesSchema = z
  .array(formulaLibraryUsageSchema)
  .nullish()
  .transform((v) => v ?? []);

/** Archiving a library templates import needs the person to confirm it. */
export type ArchiveFormulaLibraryRequest = { confirmInUse: boolean };
