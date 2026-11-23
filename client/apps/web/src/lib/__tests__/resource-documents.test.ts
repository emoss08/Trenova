import type { Document } from "@trenova/shared/types/document";
import { describe, expect, it, vi } from "vitest";
import {
  hasPendingDocumentWork,
  PENDING_DOCUMENTS_REFETCH_MS,
  PREVIEW_PENDING_MAX_POLLS,
  previewPendingRefetchMs,
  resourceDocumentsQuery,
} from "../resource-documents";

const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService,
}));

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

describe("previewPendingRefetchMs", () => {
  it("reads again while any document's preview is being drawn", () => {
    expect(
      previewPendingRefetchMs({}, [null, { previewStatus: "Ready" }, { previewStatus: "Pending" }]),
    ).toBe(PENDING_DOCUMENTS_REFETCH_MS);
  });

  it("stops once every preview has settled, and never starts for rows without documents", () => {
    const query = {};
    expect(
      previewPendingRefetchMs(query, [
        { previewStatus: "Ready" },
        { previewStatus: "Failed" },
        { previewStatus: "Unsupported" },
      ]),
    ).toBe(false);
    expect(previewPendingRefetchMs(query, [null, undefined])).toBe(false);
    expect(previewPendingRefetchMs(query, [])).toBe(false);
  });

  it("gives up on a preview that stays pending for forty intervals, and starts over once it settles", () => {
    const query = {};
    const pending = [{ previewStatus: "Pending" as const }];
    const limit = PENDING_DOCUMENTS_REFETCH_MS * PREVIEW_PENDING_MAX_POLLS;
    expect(previewPendingRefetchMs(query, pending, 1_000)).toBe(PENDING_DOCUMENTS_REFETCH_MS);
    expect(previewPendingRefetchMs(query, pending, 1_000 + limit - 1)).toBe(
      PENDING_DOCUMENTS_REFETCH_MS,
    );
    expect(previewPendingRefetchMs(query, pending, 1_000 + limit)).toBe(false);
    expect(previewPendingRefetchMs({}, pending, 1_000 + limit)).toBe(PENDING_DOCUMENTS_REFETCH_MS);

    expect(previewPendingRefetchMs(query, [{ previewStatus: "Ready" }], 2_000 + limit)).toBe(false);
    expect(previewPendingRefetchMs(query, pending, 3_000 + limit)).toBe(
      PENDING_DOCUMENTS_REFETCH_MS,
    );
  });
});
