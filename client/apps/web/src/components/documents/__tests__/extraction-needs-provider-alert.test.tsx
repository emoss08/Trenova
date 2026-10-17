import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { DocumentExtractionStatus } from "@/types/document-control";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExtractionNeedsProviderAlert } from "../extraction-needs-provider-alert";

const mocks = vi.hoisted(() => ({
  allowed: true,
  status: null as DocumentExtractionStatus | null,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    extractionStatus: () => ({
      queryKey: ["documentControl", "extraction-status"],
      queryFn: async () => mocks.status,
    }),
  },
}));

afterEach(() => {
  cleanup();
  mocks.allowed = true;
  mocks.status = null;
});

function renderAlert() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ExtractionNeedsProviderAlert />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ExtractionNeedsProviderAlert", () => {
  it("tells an administrator to add a provider when Docling has none to read with", async () => {
    mocks.status = { engine: "docling", modelProviderAvailable: false, extractionAvailable: false };

    renderAlert();

    expect(await screen.findByText("Extraction needs a model provider")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add an AI provider" })).toHaveAttribute(
      "href",
      "/admin/agent-control",
    );
  });

  it("asks someone who cannot manage providers to ask an administrator", async () => {
    mocks.allowed = false;
    mocks.status = { engine: "docling", modelProviderAvailable: false, extractionAvailable: false };

    renderAlert();

    expect(
      await screen.findByText("Ask an administrator to add an AI provider."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("stays hidden on the legacy engine even without a provider", async () => {
    mocks.status = { engine: "legacy", modelProviderAvailable: false, extractionAvailable: true };

    const { container } = renderAlert();

    await vi.waitFor(() => expect(container).toBeEmptyDOMElement());
    expect(screen.queryByText("Extraction needs a model provider")).not.toBeInTheDocument();
  });
});
