import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MemoryTab from "../memory-tab";

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({ name }: { name: string }) => <section aria-label={`${name} table`} />,
}));

vi.mock("../memory-panel", () => ({
  MemoryPanel: ({
    open,
    preset,
  }: {
    open: boolean;
    preset?: { kind?: string; content?: string };
  }) =>
    open ? (
      <aside aria-label="New memory">
        {preset?.kind}:{preset?.content}
      </aside>
    ) : null,
}));

const api = vi.hoisted(() => ({
  fetchAgentMemoryTotal: vi.fn(),
  fetchAgentMemoryUsage: vi.fn(),
  fetchAgentMemorySuggestions: vi.fn(),
  fetchRecentAgentReflections: vi.fn(),
  fetchAIRetrievalStatus: vi.fn(),
  fetchAgentControl: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-memories")>()),
  fetchAgentMemoryTotal: api.fetchAgentMemoryTotal,
  fetchAgentMemoryUsage: api.fetchAgentMemoryUsage,
  fetchAgentMemorySuggestions: api.fetchAgentMemorySuggestions,
}));
vi.mock("@/lib/graphql/agent-reflections", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-reflections")>()),
  fetchRecentAgentReflections: api.fetchRecentAgentReflections,
}));
vi.mock("@/lib/graphql/ai-retrieval", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/ai-retrieval")>()),
  fetchAIRetrievalStatus: api.fetchAIRetrievalStatus,
}));
vi.mock("@/lib/graphql/agent-control", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-control")>()),
  agentControlQueryOptions: () => ({
    queryKey: ["agent-control"],
    queryFn: () => api.fetchAgentControl(),
  }),
}));

function retrieval(activeModelKey: string | null) {
  return { settings: { activeModelKey }, sources: [], availability: { available: true } };
}

function renderTab(onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <NuqsTestingAdapter onUrlUpdate={onUrlUpdate}>
      <QueryClientProvider client={client}>
        <MemoryTab />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

beforeEach(() => {
  api.fetchAgentMemoryTotal.mockResolvedValue(3);
  api.fetchAgentMemoryUsage.mockResolvedValue({ activeCount: 3, activeSoftCap: 200, warnAt: 160 });
  api.fetchAgentMemorySuggestions.mockResolvedValue([]);
  api.fetchRecentAgentReflections.mockResolvedValue([]);
  api.fetchAIRetrievalStatus.mockResolvedValue(retrieval("openai:text-embedding-3-small"));
  api.fetchAgentControl.mockResolvedValue({ learningOff: false });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MemoryTab", () => {
  it("shows the memories table once anything is recorded, retired ones included", async () => {
    renderTab();

    expect(await screen.findByRole("region", { name: "Memory table" })).toBeInTheDocument();
    expect(screen.queryByText("Nothing recorded yet")).not.toBeInTheDocument();
  });

  it("says memories are found by their words, and links to retrieval, while nothing embeds", async () => {
    api.fetchAIRetrievalStatus.mockResolvedValue(retrieval(null));
    const onUrlUpdate = vi.fn<OnUrlUpdateFunction>();
    renderTab(onUrlUpdate);

    await userEvent.click(await screen.findByRole("button", { name: "search by meaning" }));
    expect(onUrlUpdate).toHaveBeenCalled();
    expect(onUrlUpdate.mock.calls.at(-1)?.[0].searchParams.get("tab")).toBe("retrieval");
  });

  it("does not mention search by meaning once memories are embedded", async () => {
    renderTab();

    await screen.findByRole("region", { name: "Memory table" });
    await vi.waitFor(() => expect(api.fetchAIRetrievalStatus).toHaveBeenCalled());
    expect(screen.queryByRole("button", { name: "search by meaning" })).not.toBeInTheDocument();
  });

  it("says agents won't suggest memories while learning is off", async () => {
    api.fetchAgentControl.mockResolvedValue({ learningOff: true });
    renderTab();

    expect(
      await screen.findByText(/Learning is off, so agents won't suggest new ones\./),
    ).toBeInTheDocument();
  });

  it("offers examples to start from when nothing is recorded, opening the editor with one", async () => {
    api.fetchAgentMemoryTotal.mockResolvedValue(0);
    renderTab();

    expect(await screen.findByText("Nothing recorded yet")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Memory table" })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /The receiving dock at/ }));
    expect(screen.getByRole("complementary", { name: "New memory" })).toHaveTextContent(
      "Fact:The receiving dock at … closes at",
    );
  });
});
