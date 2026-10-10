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
  reviewAgentMemory: vi.fn(),
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
    taints: false,
    reviewedAt: null,
    reviewedByUserId: null,
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
  api.reviewAgentMemory.mockImplementation(async (id) =>
    memory({ id, tainted: true, taints: false, reviewedAt: 1_791_600_000, version: 5 }),
  );
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

  it("keeps a memory written after outside content at the version that was read", async () => {
    const { onOpenChange } = renderPanel({
      mode: "edit",
      row: memory({ id: "amem_7", tainted: true, taints: true, version: 9 }),
    });

    expect(screen.getByText(/Not reviewed; every turn that reads it waits/)).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Reviewed, keep it" }));

    await waitFor(() => expect(api.reviewAgentMemory).toHaveBeenCalledWith("amem_7", 9));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("offers no review for one already reviewed, and says when it was", () => {
    renderPanel({
      mode: "edit",
      row: memory({ tainted: true, taints: false, reviewedAt: 1_791_600_000 }),
    });

    expect(screen.queryByRole("button", { name: "Reviewed, keep it" })).toBeNull();
    expect(screen.getByText(/^Reviewed .*no longer holds the writes/)).toBeTruthy();
  });

  it("keeps the review out of reach while the text has unsaved changes", async () => {
    renderPanel({ mode: "edit", row: memory({ tainted: true, taints: true }) });

    await userEvent.type(screen.getByRole("textbox", { name: "Memory" }), " Edited.");

    expect(
      (screen.getByRole("button", { name: "Reviewed, keep it" }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(api.reviewAgentMemory).not.toHaveBeenCalled();
  });

  it("shows a refused review in the editor", async () => {
    api.reviewAgentMemory.mockRejectedValue(new Error("AgentMemory has been changed"));
    renderPanel({ mode: "edit", row: memory({ tainted: true, taints: true }) });

    await userEvent.click(screen.getByRole("button", { name: "Reviewed, keep it" }));

    expect((await screen.findByRole("alert")).textContent).toContain("has been changed");
  });
});
