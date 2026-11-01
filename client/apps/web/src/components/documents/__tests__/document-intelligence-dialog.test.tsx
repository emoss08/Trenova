import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { documentSchema, type Document } from "@trenova/shared/types/document";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentIntelligenceDialog } from "../document-intelligence-dialog";

const { requestGraphQL } = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return { ...actual, requestGraphQL };
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

/** The documentContent query's answer, as the DocumentContent operation selects it. */
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
    pages: null,
  };
}

function apiError(status: number) {
  const type =
    status === 404
      ? "https://trenova.app/problems/resource-not-found"
      : "https://trenova.app/problems/internal-server-error";
  const code = status === 404 ? "NOT_FOUND" : "SYSTEM_ERROR";
  const message = status === 404 ? "Document content not found" : "Internal Server Error";
  return new GraphQLRequestError({
    kind: "graphql",
    message,
    status: 200,
    graphQLErrors: [{ extensions: { code, type }, code, message, type }],
  });
}

function respond(content: () => unknown) {
  requestGraphQL.mockImplementation(
    async ({
      operationName,
      variables,
    }: {
      operationName: string;
      variables: { documentId?: string };
    }) => {
      if (operationName === "DocumentContent" && variables.documentId === "doc_rc") {
        return { documentContent: content() };
      }
      throw new Error(`unexpected request ${operationName}`);
    },
  );
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

  it("keeps reading every few seconds, and keeps the text through a failed check", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      let calls = 0;
      respond(() => {
        calls += 1;
        if (calls === 2) throw apiError(500);
        return contentBody(calls === 1 ? "pending" : "accepted");
      });

      renderDialog();

      expect(await screen.findByText(TEXT, { normalizer: (text) => text })).toBeInTheDocument();
      expect(screen.getByText("Reading")).toBeInTheDocument();

      await vi.advanceTimersByTimeAsync(5000);
      expect(requestGraphQL).toHaveBeenCalledTimes(2);
      expect(screen.getByText(TEXT, { normalizer: (text) => text })).toBeInTheDocument();
      expect(screen.queryByText("The extracted text could not be loaded.")).toBeNull();

      await vi.advanceTimersByTimeAsync(5000);
      expect(requestGraphQL).toHaveBeenCalledTimes(3);
      expect(await screen.findByText("Accepted")).toBeInTheDocument();
      expect(screen.getByText(TEXT, { normalizer: (text) => text })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
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
