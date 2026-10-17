import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { Document } from "@trenova/shared/types/document";
import type { DocumentExtractionStatus } from "@/types/document-control";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ShipmentRateConfirmationImportPanel } from "../shipment-rate-confirmation-import-panel";

const mocks = vi.hoisted(() => ({
  calls: [] as string[],
  status: null as DocumentExtractionStatus | null,
  document: null as Document | null,
  onSuccess: null as ((document: { id: string }) => void) | null,
}));

vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: (options: { onSuccess: (document: { id: string }) => void }) => {
    mocks.onSuccess = options.onSuccess;
    return {
      uploads: [],
      uploadFiles: vi.fn(),
      cancelUpload: vi.fn(),
      retryUpload: vi.fn(),
      removeUpload: vi.fn(),
      clearAll: vi.fn(),
    };
  },
}));

vi.mock("@/hooks/use-can-add-ai-provider", () => ({
  useCanAddAIProvider: () => true,
}));

vi.mock("@/components/documents/document-shipment-draft-review-dialog", () => ({
  DocumentShipmentDraftReviewDialog: () => null,
}));

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    extractionStatus: () => ({
      queryKey: ["documentControl", "extraction-status"],
      queryFn: async () => {
        mocks.calls.push("extraction-status");
        return mocks.status;
      },
    }),
  },
}));

vi.mock("@/services/api", () => ({
  apiService: {
    documentService: {
      getById: async () => mocks.document,
      getShipmentDraft: async () => null,
      reextract: async () => {
        mocks.calls.push("reextract");
      },
    },
  },
}));

const BLOCKED: Document = {
  id: "doc_1",
  originalName: "ratecon.pdf",
  contentStatus: "Indexed",
  shipmentDraftStatus: "Unavailable",
  hasExtractedText: false,
  processingProfile: "rate_confirmation_import",
} as Document;

beforeEach(() => {
  mocks.calls = [];
  mocks.status = {
    engine: "docling",
    modelProviderAvailable: false,
    converterAvailable: true,
    extractionAvailable: false,
  };
  mocks.document = BLOCKED;
});

afterEach(() => {
  cleanup();
  mocks.onSuccess = null;
});

async function renderBlockedImport() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ShipmentRateConfirmationImportPanel open onOpenChange={() => undefined} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  await waitFor(() => expect(mocks.onSuccess).not.toBeNull());
  act(() => mocks.onSuccess?.({ id: "doc_1" }));

  await screen.findByRole("button", { name: "Retry extraction" });
}

describe("ShipmentRateConfirmationImportPanel", () => {
  it("keeps retry and replace under the provider alert for a document no provider could read", async () => {
    await renderBlockedImport();

    expect(screen.getAllByText("Extraction needs a model provider").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Retry extraction" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Replace file" })).toBeEnabled();
  });

  it("reads the extraction status again before it asks for a new extraction", async () => {
    await renderBlockedImport();
    mocks.calls = [];
    mocks.status = {
      engine: "docling",
      modelProviderAvailable: true,
      converterAvailable: true,
      extractionAvailable: true,
    };

    fireEvent.click(screen.getByRole("button", { name: "Retry extraction" }));

    await waitFor(() => expect(mocks.calls).toContain("reextract"));
    expect(mocks.calls.indexOf("extraction-status")).toBeGreaterThanOrEqual(0);
    expect(mocks.calls.indexOf("extraction-status")).toBeLessThan(mocks.calls.indexOf("reextract"));
  });
});
