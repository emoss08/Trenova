import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { documentSchema, type Document } from "@trenova/shared/types/document";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentIntelligenceDialog } from "../document-intelligence-dialog";

const { get } = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get } };
});

afterEach(cleanup);
beforeEach(() => vi.clearAllMocks());

const TEXT = "RATE CONFIRMATION\nLoad 88213\nPickup: Dallas, TX";

function rateCon(): Document {
  return documentSchema.parse({
    id: "doc_rc",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    lineageId: "doc_rc",
    versionNumber: 1,
    isCurrentVersion: true,
    fileName: "rc.pdf",
    originalName: "rate-con-88213.pdf",
    fileSize: 4096,
    fileType: "application/pdf",
    storagePath: "org_1/bu_1/rc.pdf",
    status: "Active",
    resourceId: "athr_1",
    resourceType: "assistant_thread",
    isPublic: false,
    uploadedById: "usr_1",
    previewStatus: "Ready",
    contentStatus: "Indexed",
    hasExtractedText: true,
    shipmentDraftStatus: "Unavailable",
    processingProfile: "rate_confirmation_import",
    signatureStatus: "Unknown",
    signedAt: null,
    signatureSource: "",
    version: 3,
    createdAt: 1_792_000_000,
    updatedAt: 1_792_000_030,
  });
}

/** The body of GET /documents/:id/content/ as documentcontent.Content marshals it. */
function contentBody(acceptanceStatus: string) {
  return {
    id: "dcon_rc",
    documentId: "doc_rc",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    status: "Indexed",
    contentText: TEXT,
    pageCount: 1,
    sourceKind: "native",
    detectedLanguage: "en",
    detectedDocumentKind: "RateConfirmation",
    classificationConfidence: 0.9,
    structuredData: { schemaVersion: 1, aiDiagnostics: { acceptanceStatus } },
    failureCode: "",
    failureMessage: "",
    version: 2,
    createdAt: 1_792_000_000,
    updatedAt: 1_792_000_030,
    lastExtractedAt: Math.floor(Date.now() / 1000),
    converter: "docling",
    converterVersion: "2.31.0",
  };
}

function apiError(status: number) {
  return new ApiRequestError(status, {
    type: status === 404 ? "resource-not-found" : "internal-server-error",
    title: status === 404 ? "Not Found" : "Internal Server Error",
    status,
  });
}

function respond(content: () => unknown) {
  get.mockImplementation(async (endpoint: string) => {
    if (endpoint === "/documents/doc_rc/content/") return content();
    throw new Error(`unexpected request ${endpoint}`);
  });
}

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <DocumentIntelligenceDialog
          open
          onOpenChange={vi.fn()}
          document={rateCon()}
          resourceType="assistant_thread"
          resourceId="athr_1"
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("DocumentIntelligenceDialog", () => {
  it("shows the extracted text while the model is still reading the document", async () => {
    respond(() => contentBody("pending"));

    renderDialog();

    expect(await screen.findByText(TEXT, { normalizer: (text) => text })).toBeInTheDocument();
    expect(screen.getByText("Reading")).toBeInTheDocument();
  });

  it("says the text could not be loaded and loads it again when asked", async () => {
    let calls = 0;
    respond(() => {
      calls += 1;
      if (calls === 1) throw apiError(500);
      return contentBody("accepted");
    });

    renderDialog();

    expect(await screen.findByText("The extracted text could not be loaded.")).toBeInTheDocument();
    expect(screen.queryByText("No extracted text is available for this document.")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText(TEXT, { normalizer: (text) => text })).toBeInTheDocument();
    expect(screen.getByText("Accepted")).toBeInTheDocument();
  });

  it("says there is no text for a document that was never read", async () => {
    respond(() => {
      throw apiError(404);
    });

    renderDialog();

    expect(
      await screen.findByText("No extracted text is available for this document."),
    ).toBeInTheDocument();
    expect(screen.queryByText("The extracted text could not be loaded.")).toBeNull();
  });
});
