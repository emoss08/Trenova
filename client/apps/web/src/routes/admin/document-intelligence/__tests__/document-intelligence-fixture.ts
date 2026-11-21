import type { DocumentControl } from "@/types/document-control";

/**
 * A saved document control as `GET /document-controls/` sends it (the Go entity's JSON):
 * every switch that opens rows beneath it off, with the settings under them on, drafts
 * allowed from shipments only, and the capture settings a new organization starts with.
 */
export const savedDocumentControl: DocumentControl = {
  id: "docc_01",
  version: 3,
  createdAt: 1_760_000_000,
  updatedAt: 1_760_000_500,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  enableDocumentIntelligence: false,
  enableOcr: true,
  enableAutoClassification: false,
  enableAutoDocumentTypeAssociate: false,
  enableAutoCreateDocumentTypes: true,
  enableShipmentDraftExtraction: false,
  enableAiAssistedClassification: true,
  enableAiAssistedExtraction: true,
  shipmentDraftAllowedResources: ["shipment"],
  enableFullTextIndexing: true,
  enableCapture: true,
  captureAutoFileCoverSheets: true,
  captureRetentionDays: 30,
  captureMinAgentVersion: "",
  captureAllowAutoUpdate: true,
};
