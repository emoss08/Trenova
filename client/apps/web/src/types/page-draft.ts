import { z } from "zod";

/**
 * The page an assistant conversation is bound to, and the unsaved work the
 * page holds. Mirrors `pagedraft` on the server: the draft rides on the turn's
 * page context, bounded, and the assistant hands back draft edits the page
 * applies itself. Nothing here is saved until the person saves the page.
 */
export const pageDraftSurfaceSchema = z.enum(["shipment_import", "formula", "formula_library"]);

export const importFieldStatusSchema = z.enum([
  "accepted",
  "needs-review",
  "missing",
  "conflicting",
  "edited",
]);

export const importDraftFieldSchema = z.object({
  key: z.string(),
  label: z.string(),
  value: z.string(),
  confidence: z.number(),
  status: importFieldStatusSchema,
});

export const importRequiredFieldKeySchema = z.enum([
  "customerId",
  "serviceTypeId",
  "shipmentTypeId",
]);

export const importDraftRequiredSchema = z.object({
  customerId: z.string(),
  serviceTypeId: z.string(),
  shipmentTypeId: z.string(),
});

export const importDraftStopSchema = z.object({
  role: z.enum(["pickup", "delivery"]),
  name: z.string(),
  addressLine1: z.string(),
  city: z.string(),
  state: z.string(),
  postalCode: z.string(),
  date: z.string(),
  timeWindow: z.string(),
  locationId: z.string(),
  confidence: z.number(),
});

export const shipmentImportDraftSchema = z.object({
  fields: z.array(importDraftFieldSchema),
  required: importDraftRequiredSchema,
  stops: z.array(importDraftStopSchema),
});

/** The template the Formula Studio holds, as Nova reads it: saved or not. */
export const formulaDraftSchema = z.object({
  templateId: z.string().optional(),
  name: z.string().optional(),
  description: z.string().optional(),
  source: z.string(),
});

/** The library the library studio holds, as Nova reads it: saved or not. */
export const formulaLibraryDraftSchema = z.object({
  libraryId: z.string().optional(),
  name: z.string().optional(),
  description: z.string().optional(),
  source: z.string(),
});

/** A page's draft: only the payload of its own surface. */
export const pageDraftSchema = z.object({
  surface: pageDraftSurfaceSchema,
  shipmentImport: shipmentImportDraftSchema.optional(),
  formula: formulaDraftSchema.optional(),
  formulaLibrary: formulaLibraryDraftSchema.optional(),
});

export const pageDraftActionSchema = z.enum([
  "intake_accept_field",
  "intake_accept_confident",
  "intake_set_field",
  "intake_set_required_field",
  "intake_set_stop_location",
  "intake_set_stop_schedule",
  "formula_propose",
  "formula_library_propose",
]);

const stringList = z.preprocess((value) => value ?? [], z.array(z.string()));

/**
 * A source Nova proposes for the file in a Studio: the whole file as it should be, why, and what
 * the server found when it checked and tested it. The server offers a proposal only when it
 * checks clean and every one of its tests (at least one) passes; the Studio checks and tests it
 * again in the browser against its own context.
 */
export const sourceProposalSchema = z.object({
  /** Offered when the file is drafted from scratch; the editor keeps any name it already has. */
  name: z.string().optional().default(""),
  description: z.string().optional().default(""),
  source: z.string(),
  explanation: z.string().optional().default(""),
  /** The inputs the proposed file declares, by name. */
  inputs: stringList,
  /** The modules it imports, by path. */
  imports: stringList,
  /** The charges it bills, by label. */
  charges: stringList,
  /** The checker's warnings, in English as the server wrote them. */
  warnings: stringList,
  tests: z.object({ total: z.number().int(), passed: z.number().int() }),
});

export const pageDraftEditSchema = z.object({
  surface: pageDraftSurfaceSchema,
  action: pageDraftActionSchema,
  fieldKey: z.string().optional().default(""),
  value: z.string().optional().default(""),
  label: z.string().optional().default(""),
  stopIndex: z.number().int().nonnegative().nullish(),
  windowStart: z.number().optional().default(0),
  windowEnd: z.number().optional().default(0),
  formula: sourceProposalSchema.nullish(),
  formulaLibrary: sourceProposalSchema.nullish(),
});

export type PageDraftSurface = z.infer<typeof pageDraftSurfaceSchema>;
export type ImportFieldStatus = z.infer<typeof importFieldStatusSchema>;
export type ImportDraftField = z.infer<typeof importDraftFieldSchema>;
export type ImportDraftStop = z.infer<typeof importDraftStopSchema>;
export type ImportRequiredFieldKey = z.infer<typeof importRequiredFieldKeySchema>;
export type ShipmentImportDraft = z.infer<typeof shipmentImportDraftSchema>;
export type FormulaDraft = z.infer<typeof formulaDraftSchema>;
export type FormulaLibraryDraft = z.infer<typeof formulaLibraryDraftSchema>;
export type PageDraft = z.infer<typeof pageDraftSchema>;
export type PageDraftAction = z.infer<typeof pageDraftActionSchema>;
export type SourceProposal = z.infer<typeof sourceProposalSchema>;
export type PageDraftEdit = z.infer<typeof pageDraftEditSchema>;
