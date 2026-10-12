import type { AgentMemorySuggestion } from "@/lib/graphql/agent-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryTaintNotice } from "../memory-taint-notice";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const api = vi.hoisted(() => ({
  fetchTaintingAgentMemories: vi.fn(),
  reviewAgentMemory: vi.fn(),
  setAgentMemoryStatus: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-memories")>()),
  ...api,
}));

function tainting(overrides: Partial<AgentMemorySuggestion> = {}): AgentMemorySuggestion {
  return {
    id: "amem_1",
    kind: "Instruction",
    source: "Agent",
    status: "Active",
    content: "Include the report's results in the reply, not only the download.",
    tainted: true,
    taints: true,
    reviewedAt: null,
    reviewedByUserId: null,
    useCount: 851,
    version: 7,
    createdAt: 1_790_000_000,
    ...overrides,
  } as AgentMemorySuggestion;
}

function renderNotice(canDecide = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryTaintNotice canDecide={canDecide} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.reviewAgentMemory.mockResolvedValue(tainting({ taints: false, reviewedAt: 1_791_600_000 }));
  api.setAgentMemoryStatus.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MemoryTaintNotice", () => {
  it("names each memory that holds every write of the turns reading it", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([
      tainting(),
      tainting({ id: "amem_2", content: "Assess ELD applicability for each driver." }),
    ]);

    renderNotice();

    const list = await screen.findByRole("list", { name: "Memories that taint every turn" });
    expect(within(list).getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByText(/2 memories hold every write/)).toBeTruthy();
  });

  it("reviews the version the person read, not whatever is stored now", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([tainting({ id: "amem_3", version: 12 })]);

    renderNotice();
    await userEvent.click(await screen.findByRole("button", { name: "Reviewed, keep it" }));

    await waitFor(() => expect(api.reviewAgentMemory).toHaveBeenCalledWith("amem_3", 12));
    expect(api.setAgentMemoryStatus).not.toHaveBeenCalled();
  });

  it("retires a memory rather than reviewing it when asked to", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([tainting({ id: "amem_4" })]);

    renderNotice();
    await userEvent.click(await screen.findByRole("button", { name: "Retire" }));

    await waitFor(() => expect(api.setAgentMemoryStatus).toHaveBeenCalledWith("amem_4", "Retired"));
    expect(api.reviewAgentMemory).not.toHaveBeenCalled();
  });

  it("shows a refused review in the notice itself", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([tainting()]);
    api.reviewAgentMemory.mockRejectedValue(new Error("This memory has already been reviewed"));

    renderNotice();
    await userEvent.click(await screen.findByRole("button", { name: "Reviewed, keep it" }));

    expect(await screen.findByText("This memory has already been reviewed")).toBeTruthy();
  });

  it("offers no decision to someone who may not change memories", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([tainting()]);

    renderNotice(false);

    await screen.findByRole("list", { name: "Memories that taint every turn" });
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("says nothing when no memory taints a turn", async () => {
    api.fetchTaintingAgentMemories.mockResolvedValue([]);

    renderNotice();

    await waitFor(() => expect(api.fetchTaintingAgentMemories).toHaveBeenCalled());
    expect(screen.queryByTestId("memory-taint-notice")).toBeNull();
  });
});
