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

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: false, isLoading: false }),
}));

vi.mock("@/hooks/use-plan-restrictions", () => ({
  usePlanRestrictions: () => [],
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

describe("DocumentIntelligenceForm availability", () => {
  it("names the missing document converter", async () => {
    mocks.status = {
      modelProviderAvailable: true,
      modelProviderReason: "available",
      converterAvailable: false,
      extractionAvailable: false,
    };

    renderForm();

    expect(await screen.findByText("Extraction is not set up on this server")).toBeInTheDocument();
    expect(
      screen.getByText("Ask an administrator to configure the document converter."),
    ).toBeInTheDocument();
  });

  it("names the missing model provider when the converter is configured", async () => {
    mocks.status = {
      modelProviderAvailable: false,
      modelProviderReason: "no_model_provider",
      converterAvailable: true,
      extractionAvailable: false,
    };

    renderForm();

    expect(await screen.findByText("Extraction needs a model provider")).toBeInTheDocument();
  });

  it("shows no alert while documents can be read", async () => {
    mocks.status = {
      modelProviderAvailable: true,
      modelProviderReason: "available",
      converterAvailable: true,
      extractionAvailable: true,
    };

    renderForm();

    expect(await screen.findByText("Platform availability")).toBeInTheDocument();
    expect(screen.queryByText("Extraction is not set up on this server")).not.toBeInTheDocument();
    expect(screen.queryByText("Extraction needs a model provider")).not.toBeInTheDocument();
  });
});
