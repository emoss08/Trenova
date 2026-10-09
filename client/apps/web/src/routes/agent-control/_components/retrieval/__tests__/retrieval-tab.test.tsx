import type { AIRetrievalSource, AIRetrievalStatus } from "@/lib/graphql/ai-retrieval";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import RetrievalTab from "../retrieval-tab";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const permissions = vi.hoisted(() => ({ allowed: true }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: permissions.allowed, isLoading: false }),
}));

vi.mock("../failed-entries-table", () => ({
  default: () => <p>Failed items table</p>,
}));

const api = vi.hoisted(() => ({
  fetchAIRetrievalStatus: vi.fn(),
  updateAIRetrievalSettings: vi.fn(),
  fetchAIProviders: vi.fn(),
}));

vi.mock("@/lib/graphql/ai-retrieval", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/ai-retrieval")>()),
  fetchAIRetrievalStatus: api.fetchAIRetrievalStatus,
  updateAIRetrievalSettings: api.updateAIRetrievalSettings,
}));
vi.mock("@/lib/graphql/ai-provider", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/ai-provider")>()),
  fetchAIProviders: api.fetchAIProviders,
}));

function source(overrides: Partial<AIRetrievalSource>): AIRetrievalSource {
  return {
    sourceType: "Memory",
    enabled: true,
    total: 0,
    indexed: 0,
    pending: 0,
    failed: 0,
    skipped: 0,
    lastIndexedAt: null,
    lastAttemptAt: null,
    ...overrides,
  };
}

function status(overrides: Partial<AIRetrievalStatus> = {}): AIRetrievalStatus {
  return {
    availability: {
      available: true,
      reason: null,
      extensionInstalled: true,
      extensionVersion: "0.8.1",
    },
    settings: {
      memoryEnabled: true,
      documentsEnabled: true,
      inboundMessagesEnabled: false,
      monthlyIndexingBudgetUsd: "10.00",
      paused: false,
      pausedReason: null,
      pausedAt: null,
      activeModelKey: "nomic-embed-text@768",
      dimensions: 768,
      pendingModelKey: null,
      pendingDimensions: null,
      version: 3,
      updatedAt: 1_790_000_000,
    },
    sources: [
      source({ sourceType: "Memory", total: 6, indexed: 6 }),
      source({ sourceType: "Document", total: 100, indexed: 90, failed: 4, skipped: 6 }),
      source({ sourceType: "InboundMessage", enabled: false, total: 30 }),
    ],
    monthStartedAt: 1_788_220_800,
    indexingCostMonthUsd: "2.50",
    indexingUnpricedCalls: 0,
    retrievalCostMonthUsd: "0.10",
    retrievalUnpricedCalls: 0,
    lastIndexedAt: null,
    modelChange: null,
    configuredModelKey: "nomic-embed-text@768",
    configuredModelDiffers: false,
    ...overrides,
  } as AIRetrievalStatus;
}

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenProviders = vi.fn();
  render(
    <NuqsTestingAdapter>
      <QueryClientProvider client={client}>
        <RetrievalTab onOpenProviders={onOpenProviders} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );

  return { onOpenProviders };
}

beforeEach(() => {
  permissions.allowed = true;
  api.fetchAIRetrievalStatus.mockResolvedValue(status());
  api.updateAIRetrievalSettings.mockImplementation(async () => status());
  api.fetchAIProviders.mockResolvedValue([
    { id: "aiprv_2", name: "Backup", enabled: true, priority: 20, tasks: ["Embedding"] },
    { id: "aiprv_1", name: "Local Ollama", enabled: true, priority: 10, tasks: ["Embedding"] },
  ]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RetrievalTab", () => {
  it("says everything is searchable except what failed, and names the provider first in line", async () => {
    renderTab();

    expect(
      await screen.findByText(/Everything is searchable by meaning under/),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "4 items that failed" })).toBeInTheDocument();
    expect(await screen.findByText("Routed to Local Ollama")).toBeInTheDocument();
  });

  it("says nothing handles Embedding and sends the person to Providers", async () => {
    const base = status();
    api.fetchAIRetrievalStatus.mockResolvedValue(
      status({
        availability: { ...base.availability, available: false, reason: "NoProvider" },
        settings: { ...base.settings, activeModelKey: null },
        configuredModelKey: null,
      }),
    );
    const { onOpenProviders } = renderTab();

    expect(await screen.findByText(/Nothing handles the Embedding task/)).toBeInTheDocument();
    expect(screen.getByText("100 items are")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Route Embedding" }));
    expect(onOpenProviders).toHaveBeenCalled();
  });

  it("pauses indexing from Nova's control, sending only the pause", async () => {
    renderTab();

    await userEvent.click(await screen.findByRole("button", { name: "Pause indexing" }));
    await waitFor(() =>
      expect(api.updateAIRetrievalSettings).toHaveBeenCalledWith({ paused: true }),
    );
  });

  it("resumes a paused index", async () => {
    const base = status();
    api.fetchAIRetrievalStatus.mockResolvedValue(
      status({ settings: { ...base.settings, paused: true, pausedReason: "Manual" } }),
    );
    renderTab();

    await userEvent.click(await screen.findByRole("button", { name: "Resume indexing" }));
    await waitFor(() =>
      expect(api.updateAIRetrievalSettings).toHaveBeenCalledWith({ paused: false }),
    );
  });

  it("turns one source on or off, sending that source's setting alone", async () => {
    renderTab();

    await userEvent.click(await screen.findByRole("switch", { name: "Index Inbound email" }));
    await waitFor(() =>
      expect(api.updateAIRetrievalSettings).toHaveBeenCalledWith({ inboundMessagesEnabled: true }),
    );
  });

  it("saves a new budget when the field is left, and nothing when it is unchanged or not a budget", async () => {
    renderTab();

    const budget = await screen.findByRole("textbox", { name: "Monthly indexing budget" });
    await userEvent.click(budget);
    await userEvent.tab();
    expect(api.updateAIRetrievalSettings).not.toHaveBeenCalled();

    await userEvent.clear(budget);
    await userEvent.type(budget, "ten");
    await userEvent.tab();
    expect(api.updateAIRetrievalSettings).not.toHaveBeenCalled();

    await userEvent.clear(budget);
    await userEvent.type(budget, "25");
    await userEvent.tab();
    await waitFor(() =>
      expect(api.updateAIRetrievalSettings).toHaveBeenCalledWith({
        monthlyIndexingBudgetUsd: "25.00",
      }),
    );
  });

  it("shows a source's counts and leaves a turned-off one without a re-index", async () => {
    renderTab();

    expect(await screen.findByText("4 failed")).toBeInTheDocument();
    expect(screen.getByText("6 skipped")).toBeInTheDocument();
    const reindex = screen.getAllByRole("button", { name: "Re-index" });
    expect(reindex[2]).toBeDisabled();
    expect(reindex[0]).toBeEnabled();
  });

  it("offers no changes to someone who cannot make them", async () => {
    permissions.allowed = false;
    renderTab();

    await screen.findByText(/Everything is searchable by meaning/);
    expect(screen.queryByRole("button", { name: "Pause indexing" })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: /^Index / })).not.toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Pause indexing" })).toBeDisabled();
    expect(screen.getByRole("textbox", { name: "Monthly indexing budget" })).toBeDisabled();
  });

  it("says a missing database extension below Nova, with the command that fixes it", async () => {
    const base = status();
    api.fetchAIRetrievalStatus.mockResolvedValue(
      status({
        availability: { ...base.availability, available: false, reason: "ExtensionMissing" },
      }),
    );
    renderTab();

    const notice = await screen.findByTestId("retrieval-notice");
    expect(within(notice).getByText("trenova db enable-vector")).toBeInTheDocument();
  });
});
