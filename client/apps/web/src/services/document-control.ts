import {
  DocumentControlDocument,
  UpdateDocumentControlDocument,
  type UpdateDocumentControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { documentControlSchema, type DocumentControl } from "@/types/document-control";

export function toDocumentControlInput(data: DocumentControl): UpdateDocumentControlInput {
  return {
    version: data.version ?? 0,
    enableDocumentIntelligence: data.enableDocumentIntelligence,
    enableOcr: data.enableOcr,
    enableAutoClassification: data.enableAutoClassification,
    enableAutoDocumentTypeAssociate: data.enableAutoDocumentTypeAssociate,
    enableAutoCreateDocumentTypes: data.enableAutoCreateDocumentTypes,
    enableShipmentDraftExtraction: data.enableShipmentDraftExtraction,
    enableAiAssistedClassification: data.enableAiAssistedClassification,
    enableAiAssistedExtraction: data.enableAiAssistedExtraction,
    shipmentDraftAllowedResources: data.shipmentDraftAllowedResources,
    enableFullTextIndexing: data.enableFullTextIndexing,
    enableCapture: data.enableCapture,
    captureAutoFileCoverSheets: data.captureAutoFileCoverSheets,
    captureRetentionDays: data.captureRetentionDays,
    captureMinAgentVersion: data.captureMinAgentVersion ?? "",
    captureAllowAutoUpdate: data.captureAllowAutoUpdate,
  };
}

export class DocumentControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: DocumentControlDocument,
      operationName: "DocumentControl",
      signal: options?.signal,
    });

    return safeParse(documentControlSchema, response.documentControl, "Document Control");
  }

  public async update(data: DocumentControl) {
    const response = await requestGraphQL({
      document: UpdateDocumentControlDocument,
      operationName: "UpdateDocumentControl",
      variables: { input: toDocumentControlInput(data) },
    });

    return safeParse(documentControlSchema, response.updateDocumentControl, "Document Control");
  }
}
