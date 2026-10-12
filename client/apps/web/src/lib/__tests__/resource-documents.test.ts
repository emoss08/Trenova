import type { Document } from "@trenova/shared/types/document";
import { describe, expect, it, vi } from "vitest";
import { hasPendingDocumentWork, resourceDocumentsQuery } from "../resource-documents";

const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { documentService } }));

function document(fields: Partial<Document>): Document {
  return {
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    ...fields,
  } as Document;
}

describe("resourceDocumentsQuery", () => {
  it("reads a record's documents with their types under the key every writer refreshes", async () => {
    documentService.getByResource.mockResolvedValue([]);
    const query = resourceDocumentsQuery("shipment", "shp_1");

    expect(query.queryKey).toEqual(["documents", "shipment", "shp_1", ""]);
    await query.queryFn?.({} as never);
    expect(documentService.getByResource).toHaveBeenCalledWith("shipment", "shp_1", "", {
      includeDocumentType: "true",
    });
  });
});

describe("hasPendingDocumentWork", () => {
  it("waits on a preview the server has still to draw", () => {
    expect(hasPendingDocumentWork(document({ previewStatus: "Pending" }))).toBe(true);
    expect(hasPendingDocumentWork(document({ previewStatus: "Failed" }))).toBe(false);
  });

  it("waits on a rate confirmation still being read or drafted, and on nothing else's content", () => {
    const rateConfirmation = { processingProfile: "rate_confirmation_import" } as const;

    expect(
      hasPendingDocumentWork(document({ ...rateConfirmation, contentStatus: "Extracting" })),
    ).toBe(true);
    expect(
      hasPendingDocumentWork(document({ ...rateConfirmation, shipmentDraftStatus: "Pending" })),
    ).toBe(true);
    expect(hasPendingDocumentWork(document({ ...rateConfirmation }))).toBe(false);
    expect(hasPendingDocumentWork(document({ contentStatus: "Extracting" }))).toBe(false);
  });
});
