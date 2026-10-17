import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { DocumentControl, DocumentExtractionStatus } from "@/types/document-control";
import { MemoryRouter } from "react-router";
import { Suspense } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DocumentIntelligenceForm from "../document-intelligence-form";

const mocks = vi.hoisted(() => ({
  status: null as DocumentExtractionStatus | null,
}));

const CONTROL: DocumentControl = {
  id: "dc_1",
  version: 1,
  enableDocumentIntelligence: true,
  enableOcr: true,
  enableAutoClassification: true,
  enableAutoDocumentTypeAssociate: true,
  enableAutoCreateDocumentTypes: true,
  enableShipmentDraftExtraction: true,
  enableAiAssistedClassification: true,
  enableAiAssistedExtraction: true,
  shipmentDraftAllowedResources: ["shipment"],
  enableFullTextIndexing: true,
  enableCapture: true,
  captureAutoFileCoverSheets: true,
  captureRetentionDays: 30,
  captureMinAgentVersion: "",
  captureAllowAutoUpdate: true,
  extractionEngine: "legacy",
} as DocumentControl;

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    get: Object.assign(
      () => ({ queryKey: ["documentControl", "get"], queryFn: async () => CONTROL }),
      { _def: ["documentControl", "get"] },
    ),
    extractionStatus: Object.assign(
      () => ({
        queryKey: ["documentControl", "extraction-status"],
        queryFn: async () => mocks.status,
      }),
      { _def: ["documentControl", "extraction-status"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

afterEach(() => {
  cleanup();
  mocks.status = null;
});

function renderForm() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Suspense fallback={null}>
          <DocumentIntelligenceForm />
        </Suspense>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function doclingOption() {
  return screen.getByRole("radio", { name: "Docling" });
}

describe("DocumentIntelligenceForm extraction engine", () => {
  it("disables Docling and says why when the server has no document converter", async () => {
    mocks.status = {
      engine: "legacy",
      modelProviderAvailable: true,
      converterAvailable: false,
      extractionAvailable: true,
    };

    renderForm();

    expect(
      await screen.findByText(
        "Docling is unavailable: the document converter is not configured on this server. An administrator sets documentConversion.doclingUrl and documentConversion.apiKey to turn it on.",
      ),
    ).toBeInTheDocument();
    expect(doclingOption()).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Legacy" })).toBeEnabled();
  });

  it("offers Docling when the converter is configured", async () => {
    mocks.status = {
      engine: "legacy",
      modelProviderAvailable: true,
      converterAvailable: true,
      extractionAvailable: true,
    };

    renderForm();

    expect(await screen.findByText("Extraction engine")).toBeInTheDocument();
    await vi.waitFor(() => expect(doclingOption()).toBeEnabled());
  });
});
