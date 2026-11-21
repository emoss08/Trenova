import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

// Mirrors versionutils.Parse: MAJOR.MINOR.PATCH, an optional leading "v", no
// leading zeros and no pre-release suffix. The column holds 20 characters.
const CAPTURE_AGENT_VERSION = /^v?(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/;

export const documentControlResourceSchema = z.enum(["shipment", "trailer", "tractor", "worker"]);

export const documentControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    enableDocumentIntelligence: z.boolean(),
    enableOcr: z.boolean(),
    enableAutoClassification: z.boolean(),
    enableAutoDocumentTypeAssociate: z.boolean(),
    enableAutoCreateDocumentTypes: z.boolean(),
    enableShipmentDraftExtraction: z.boolean(),
    enableAiAssistedClassification: z.boolean(),
    enableAiAssistedExtraction: z.boolean(),
    shipmentDraftAllowedResources: z.array(documentControlResourceSchema),
    enableFullTextIndexing: z.boolean(),
    enableCapture: z.boolean(),
    captureAutoFileCoverSheets: z.boolean(),
    captureRetentionDays: z
      .number({ error: () => translate("Enter how many days to keep unfiled pages") })
      .int()
      .min(1, { error: () => translate("Keep unfiled pages for at least 1 day") })
      .max(365, { error: () => translate("Keep unfiled pages for at most 365 days") }),
    captureMinAgentVersion: z
      .string()
      .trim()
      .max(20, { error: () => translate("Use a version like 1.4.0") })
      .refine((value) => value === "" || CAPTURE_AGENT_VERSION.test(value), {
        error: () => translate("Use a version like 1.4.0"),
      })
      .optional(),
    captureAllowAutoUpdate: z.boolean(),
  })
  .superRefine((values, ctx) => {
    // Drafts that may come from no record are never made: the switch would be on and do
    // nothing. The list is kept as it was while drafting is off.
    if (values.enableShipmentDraftExtraction && values.shipmentDraftAllowedResources.length === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["shipmentDraftAllowedResources"],
        message: translate("Choose at least one kind of record to draft from"),
      });
    }
  });

export const extractionModelReasonSchema = z.enum([
  "available",
  "off_on_server",
  "not_in_plan",
  "no_model_provider",
]);

export const documentExtractionStatusSchema = z.object({
  modelProviderAvailable: z.boolean(),
  modelProviderReason: extractionModelReasonSchema.optional(),
  converterAvailable: z.boolean(),
  extractionAvailable: z.boolean(),
});

export const documentStorageStatusSchema = z.object({
  managed: z.boolean(),
  used: z.number(),
  included: z.number(),
  overage: z.boolean(),
  warning: z.boolean(),
  stopped: z.boolean(),
});

export type DocumentControl = z.infer<typeof documentControlSchema>;

/**
 * The settings of the record that belong to Trenova Capture. The Scanning and printing page
 * edits them; the Document extraction page sends them back as it read them.
 */
export const captureSettingFields = [
  "enableCapture",
  "captureAutoFileCoverSheets",
  "captureRetentionDays",
  "captureMinAgentVersion",
  "captureAllowAutoUpdate",
] as const satisfies readonly (keyof DocumentControl)[];
export type DocumentControlResource = z.infer<typeof documentControlResourceSchema>;
export type ExtractionModelReason = z.infer<typeof extractionModelReasonSchema>;
export type DocumentExtractionStatus = z.infer<typeof documentExtractionStatusSchema>;
export type DocumentStorageStatus = z.infer<typeof documentStorageStatusSchema>;
