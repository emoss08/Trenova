import { z } from "zod";

/**
 * The page an assistant conversation is bound to, and the unsaved work the
 * page holds. Mirrors `pagedraft` on the server: the draft rides on the turn's
 * page context, bounded, and the assistant hands back draft edits the page
 * applies itself. Nothing here is saved until the person saves the page.
 */
export const pageDraftSurfaceSchema = z.enum(["shipment_import", "formula"]);

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
  "formulaTemplateId",
]);

export const importDraftRequiredSchema = z.object({
  customerId: z.string(),
  serviceTypeId: z.string(),
  shipmentTypeId: z.string(),
  formulaTemplateId: z.string(),
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

export const formulaDraftVariableTypeSchema = z.enum(["Number", "String", "Boolean"]);

export const formulaDraftVariableSchema = z.object({
  name: z.string(),
  type: formulaDraftVariableTypeSchema,
  description: z.string().optional().default(""),
  defaultValue: z.union([z.string(), z.number(), z.boolean()]).nullish(),
});

export const formulaDraftSchema = z.object({
  templateId: z.string().optional(),
  schemaId: z.string(),
  templateType: z.string(),
  expression: z.string(),
  variables: z.array(formulaDraftVariableSchema),
});

export const pageDraftSchema = z.object({
  surface: pageDraftSurfaceSchema,
  shipmentImport: shipmentImportDraftSchema.optional(),
  formula: formulaDraftSchema.optional(),
});

export const pageDraftActionSchema = z.enum([
  "accept_field",
  "accept_all_confident",
  "set_field_value",
  "set_required_field",
  "set_stop_location",
  "set_stop_schedule",
  "propose_formula",
]);

export const formulaCheckSchema = z.object({
  valid: z.boolean(),
  result: z.string().optional().default(""),
  error: z.string().optional().default(""),
});

/** A sample load the formula engine priced; `amount` is the engine's, never the model's. */
export const pricedScenarioSchema = z.object({
  name: z.string(),
  description: z.string().optional().default(""),
  variables: z.preprocess((value) => value ?? {}, z.record(z.string(), z.unknown())),
  amount: z.string().optional().default(""),
  valid: z.boolean(),
  error: z.string().optional().default(""),
});

export const formulaProposalSchema = z.object({
  schemaId: z.string(),
  expression: z.string(),
  variables: z.preprocess((value) => value ?? [], z.array(formulaDraftVariableSchema)),
  explanation: z.string().optional().default(""),
  check: formulaCheckSchema,
  scenarios: z.preprocess((value) => value ?? [], z.array(pricedScenarioSchema)),
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
  formula: formulaProposalSchema.nullish(),
});

export type PageDraftSurface = z.infer<typeof pageDraftSurfaceSchema>;
export type ImportFieldStatus = z.infer<typeof importFieldStatusSchema>;
export type ImportDraftField = z.infer<typeof importDraftFieldSchema>;
export type ImportDraftStop = z.infer<typeof importDraftStopSchema>;
export type ImportRequiredFieldKey = z.infer<typeof importRequiredFieldKeySchema>;
export type ShipmentImportDraft = z.infer<typeof shipmentImportDraftSchema>;
export type FormulaDraftVariable = z.infer<typeof formulaDraftVariableSchema>;
export type FormulaDraft = z.infer<typeof formulaDraftSchema>;
export type PageDraft = z.infer<typeof pageDraftSchema>;
export type PageDraftAction = z.infer<typeof pageDraftActionSchema>;
export type PricedScenario = z.infer<typeof pricedScenarioSchema>;
export type FormulaProposal = z.infer<typeof formulaProposalSchema>;
export type PageDraftEdit = z.infer<typeof pageDraftEditSchema>;
