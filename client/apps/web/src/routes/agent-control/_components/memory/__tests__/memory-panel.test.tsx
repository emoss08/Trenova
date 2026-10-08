import type { AgentMemoryRow } from "@/lib/graphql/agent-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryPanel } from "../memory-panel";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const api = vi.hoisted(() => ({
  createAgentMemory: vi.fn(),
  updateAgentMemory: vi.fn(),
  setAgentMemoryStatus: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-memories")>()),
  ...api,
}));

function memory(overrides: Partial<AgentMemoryRow> = {}): AgentMemoryRow {
  return {
    id: "amem_1",
    kind: "Fact",
    source: "User",
    status: "Active",
    subjectType: null,
    subjectId: null,
    subjectLabel: "",
    toolName: "",
    content: "Dock 4 closes at 15:00 on Fridays.",
    tainted: false,
    useCount: 6,
    createdAt: 1_790_000_000,
    expiresAt: null,
    lastUsedAt: null,
    version: 4,
    evidence: null,
    supersedes: null,
    replacedBy: null,
    ...overrides,
  } as AgentMemoryRow;
}

function renderPanel(props: { mode: "create" | "edit"; row: AgentMemoryRow | null }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <MemoryPanel open onOpenChange={onOpenChange} {...props} />
    </QueryClientProvider>,
  );

  return { onOpenChange };
}

beforeEach(() => {
  api.createAgentMemory.mockImplementation(async (input) =>
    memory({ ...input, id: "amem_9", version: 0 }),
  );
  api.updateAgentMemory.mockImplementation(async (id, input) =>
    memory({ ...input, id, version: 5 }),
  );
  api.setAgentMemoryStatus.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MemoryPanel", () => {
  it("creates a memory every agent reads, sending nothing it left unset", async () => {
    renderPanel({ mode: "create", row: null });

    await userEvent.type(
      screen.getByRole("textbox", { name: "Memory" }),
      "Always copy ap@acme.com on Acme invoices.",
    );
    await userEvent.click(screen.getByRole("button", { name: /Save memory/ }));

    await waitFor(() =>
      expect(api.createAgentMemory).toHaveBeenCalledWith({
        kind: "Instruction",
        content: "Always copy ap@acme.com on Acme invoices.",
        subjectType: null,
        subjectId: null,
        toolName: null,
        expiresAt: null,
        version: 0,
      }),
    );
  });

  it("saves a kept memory at the version it was read at", async () => {
    renderPanel({ mode: "edit", row: memory() });

    const box = screen.getByRole("textbox", { name: "Memory" });
    await userEvent.clear(box);
    await userEvent.type(box, "Dock 4 closes at 14:00 on Fridays.");
    await userEvent.click(screen.getByRole("button", { name: /Save changes/ }));

    await waitFor(() =>
      expect(api.updateAgentMemory).toHaveBeenCalledWith(
        "amem_1",
        expect.objectContaining({ content: "Dock 4 closes at 14:00 on Fridays.", version: 4 }),
      ),
    );
  });

  it("retires a kept memory and closes", async () => {
    const { onOpenChange } = renderPanel({ mode: "edit", row: memory() });

    await userEvent.click(screen.getByRole("button", { name: "Retire memory" }));

    await waitFor(() => expect(api.setAgentMemoryStatus).toHaveBeenCalledWith("amem_1", "Retired"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("restores a retired memory", async () => {
    renderPanel({ mode: "edit", row: memory({ status: "Retired" }) });

    await userEvent.click(screen.getByRole("button", { name: "Restore memory" }));

    await waitFor(() => expect(api.setAgentMemoryStatus).toHaveBeenCalledWith("amem_1", "Active"));
  });
});
