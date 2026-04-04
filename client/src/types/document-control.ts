import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const documentControlResourceSchema = z.enum([
  "shipment",
  "trailer",
  "tractor",
  "worker",
]);

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
});

export type DocumentControl = z.infer<typeof documentControlSchema>;
export type DocumentControlResource = z.infer<
  typeof documentControlResourceSchema
>;
