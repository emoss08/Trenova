import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";

// Mirrors versionutils.Parse: MAJOR.MINOR.PATCH, an optional leading "v", no
// leading zeros and no pre-release suffix. The column holds 20 characters.
const CAPTURE_AGENT_VERSION = /^v?(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/;

export const documentControlResourceSchema = z.enum(["shipment", "trailer", "tractor", "worker"]);

export const documentControlSchema = z.object({
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
  captureRetentionDays: z.number().int().min(1).max(365),
  captureMinAgentVersion: z
    .string()
    .trim()
    .max(20, { error: "Use a version like 1.4.0" })
    .refine((value) => value === "" || CAPTURE_AGENT_VERSION.test(value), {
      error: "Use a version like 1.4.0",
    })
    .optional(),
  captureAllowAutoUpdate: z.boolean(),
});

export type DocumentControl = z.infer<typeof documentControlSchema>;
export type DocumentControlResource = z.infer<typeof documentControlResourceSchema>;
