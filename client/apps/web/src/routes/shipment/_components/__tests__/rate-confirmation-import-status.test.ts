import type { Document, DocumentShipmentDraft } from "@trenova/shared/types/document";
import { describe, expect, it } from "vitest";
import { isBlockedByProvider, processingSummary } from "../rate-confirmation-import-status";

function storedWithoutText(overrides: Partial<Document> = {}): Document {
  return {
    id: "doc_1",
    originalName: "ratecon.pdf",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    hasExtractedText: false,
    processingProfile: "rate_confirmation_import",
    ...overrides,
  } as Document;
}

describe("isBlockedByProvider", () => {
  it("holds a stored, unread document when no provider can read documents", () => {
    expect(isBlockedByProvider(storedWithoutText(), false)).toBe(true);
  });

  it("never holds a document while extraction is available", () => {
    expect(isBlockedByProvider(storedWithoutText(), true)).toBe(false);
  });

  it("does not blame the provider for a document whose text was read", () => {
    expect(isBlockedByProvider(storedWithoutText({ hasExtractedText: true }), false)).toBe(false);
  });

  it("does not blame the provider while the document is still being read", () => {
    expect(isBlockedByProvider(storedWithoutText({ contentStatus: "Extracting" }), false)).toBe(
      false,
    );
  });

  it("does not blame the provider before the upload has a document", () => {
    expect(isBlockedByProvider(undefined, false)).toBe(false);
    expect(isBlockedByProvider(null, false)).toBe(false);
  });
});

describe("processingSummary", () => {
  it("says extraction needs a model provider when none can read documents", () => {
    expect(processingSummary(storedWithoutText(), null, false)).toEqual({
      title: "Extraction needs a model provider",
      description:
        "The file is stored, but no AI provider can read documents for your organization, so no shipment draft was built.",
      progress: 100,
      variant: "error",
    });
  });

  it("keeps the no-draft summary when extraction was available", () => {
    expect(processingSummary(storedWithoutText(), null, true)).toEqual({
      title: "No shipment draft available",
      description: "This file did not produce a usable shipment draft.",
      progress: 100,
      variant: "error",
    });
  });

  it("keeps the no-draft summary when extraction is assumed available", () => {
    expect(processingSummary(storedWithoutText(), null).title).toBe("No shipment draft available");
  });

  it("reports a ready draft ahead of provider availability", () => {
    const draft = { status: "Ready" } as DocumentShipmentDraft;

    expect(processingSummary(storedWithoutText(), draft, false).title).toBe("Shipment draft ready");
  });
});
