import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { DocumentExtractionStatus } from "@/types/document-control";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExtractionNeedsProviderAlert } from "../extraction-needs-provider-alert";

const mocks = vi.hoisted(() => ({
  granted: new Set<string>(),
  restrictions: [] as string[],
  status: null as DocumentExtractionStatus | null,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@/hooks/use-plan-restrictions", () => ({
  usePlanRestrictions: () => mocks.restrictions,
}));

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    extractionStatus: () => ({
      queryKey: ["documentControl", "extraction-status"],
      queryFn: async () => mocks.status,
    }),
  },
}));

const NO_PROVIDER: DocumentExtractionStatus = {
  engine: "docling",
  modelProviderAvailable: false,
  modelProviderReason: "no_model_provider",
  converterAvailable: true,
  extractionAvailable: false,
};

function grantProviderSetup() {
  mocks.granted.add(`${Resource.AIProvider}:${Operation.Create}`);
  mocks.granted.add(`${Resource.AgentControl}:${Operation.Read}`);
}

afterEach(() => {
  cleanup();
  mocks.granted.clear();
  mocks.restrictions = [];
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
  it("links someone who can add a provider to the providers tab of AI control", async () => {
    grantProviderSetup();
    mocks.status = NO_PROVIDER;

    renderAlert();

    expect(await screen.findByText("Extraction needs a model provider")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add an AI provider" })).toHaveAttribute(
      "href",
      "/admin/agent-control?tab=providers",
    );
  });

  it("asks someone who can only open AI control to ask an administrator", async () => {
    mocks.granted.add(`${Resource.AgentControl}:${Operation.Read}`);
    mocks.status = NO_PROVIDER;

    renderAlert();

    expect(
      await screen.findByText("Ask an administrator to add an AI provider."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("hides the link when the plan withholds agent automation", async () => {
    grantProviderSetup();
    mocks.restrictions = ["agent.automation"];
    mocks.status = NO_PROVIDER;

    renderAlert();

    expect(
      await screen.findByText("Ask an administrator to add an AI provider."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("says the converter is not configured rather than asking for a provider", async () => {
    grantProviderSetup();
    mocks.status = {
      engine: "docling",
      modelProviderAvailable: true,
      converterAvailable: false,
      extractionAvailable: false,
    };

    renderAlert();

    expect(await screen.findByText("Extraction is not set up on this server")).toBeInTheDocument();
    expect(
      screen.getByText("Ask an administrator to configure the document converter."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Extraction needs a model provider")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("names the server switch rather than asking for a provider when extraction is off", async () => {
    grantProviderSetup();
    mocks.status = { ...NO_PROVIDER, modelProviderReason: "off_on_server" };

    renderAlert();

    expect(await screen.findByText("Extraction is turned off on this server")).toBeInTheDocument();
    expect(
      screen.getByText("Ask an administrator to set ai.documentExtraction to true."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Extraction needs a model provider")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("says the plan leaves extraction out rather than asking for a provider", async () => {
    grantProviderSetup();
    mocks.status = { ...NO_PROVIDER, modelProviderReason: "not_in_plan" };

    renderAlert();

    expect(await screen.findByText("Extraction is not in your plan")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Add an AI provider" })).not.toBeInTheDocument();
  });

  it("treats a status without a reason as a missing provider", async () => {
    grantProviderSetup();
    mocks.status = { ...NO_PROVIDER, modelProviderReason: undefined };

    renderAlert();

    expect(await screen.findByText("Extraction needs a model provider")).toBeInTheDocument();
  });

  it("stays hidden on the legacy engine even without a provider", async () => {
    mocks.status = {
      engine: "legacy",
      modelProviderAvailable: false,
      converterAvailable: false,
      extractionAvailable: true,
    };

    const { container } = renderAlert();

    await vi.waitFor(() => expect(container).toBeEmptyDOMElement());
    expect(screen.queryByText("Extraction needs a model provider")).not.toBeInTheDocument();
  });
});
