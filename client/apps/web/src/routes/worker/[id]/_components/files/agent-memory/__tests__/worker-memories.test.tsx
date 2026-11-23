import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerMemories from "../worker-memories";

const mocks = vi.hoisted(() => ({
  fetchDeskMemories: vi.fn(),
  setDeskMemoryStatus: vi.fn(),
  createDeskMemory: vi.fn(),
  available: true,
}));

vi.mock("@/lib/graphql/desk-memories", () => ({
  DESK_MEMORIES_KEY: "desk-memories",
  fetchDeskMemories: mocks.fetchDeskMemories,
  setDeskMemoryStatus: mocks.setDeskMemoryStatus,
  createDeskMemory: mocks.createDeskMemory,
  reviseDeskMemory: vi.fn(),
}));

vi.mock("@/components/record-memories/record-memories", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/components/record-memories/record-memories")>();
  return { ...actual, useRecordMemoriesAvailable: () => mocks.available };
});

function memory(overrides: Record<string, unknown>) {
  return {
    id: "mem",
    content: "Note",
    kind: "Fact",
    scope: "Organization",
    roleId: null,
    roleName: null,
    subjectType: "Worker",
    subjectLabel: "Maria Alvarez",
    status: "Active",
    source: "Agent",
    sourceTitle: "Load planning",
    useCount: 0,
    lastUsedAt: null,
    createdAt: 1_792_000_000,
    version: 1,
    editable: true,
    reason: null,
    quotes: null,
    replaces: null,
    replacedBy: null,
    ...overrides,
  };
}

function renderView() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkerMemories workerId="wrk_1" workerName="Maria" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.available = true;
});

describe("WorkerMemories", () => {
  it("lists what agents remember about the worker with a count", async () => {
    mocks.fetchDeskMemories.mockResolvedValue({
      items: [
        memory({
          id: "mem_1",
          content: "Prefers loads that get her home through Joliet on Fridays.",
        }),
        memory({
          id: "mem_2",
          content: "Will not take loads into New York City.",
          source: "User",
          sourceTitle: "",
        }),
      ],
    });
    renderView();

    expect(await screen.findByText("What agents remember about Maria")).toBeTruthy();
    expect(screen.getByText("2 notes")).toBeTruthy();
    expect(screen.getByText("Will not take loads into New York City.")).toBeTruthy();
    expect(mocks.fetchDeskMemories).toHaveBeenCalledWith(
      { scope: null, roleId: null, query: "", subject: { type: "Worker", id: "wrk_1" } },
      null,
      50,
      expect.anything(),
    );
  });

  it("removes a note only after the confirmation", async () => {
    mocks.fetchDeskMemories.mockResolvedValue({ items: [memory({ id: "mem_1" })] });
    mocks.setDeskMemoryStatus.mockResolvedValue(memory({ id: "mem_1", status: "Retired" }));
    renderView();
    const user = userEvent.setup();

    const row = within(await screen.findByTestId("memory-mem_1"));
    await user.click(row.getByRole("button", { name: "Remove this note" }));
    expect(mocks.setDeskMemoryStatus).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Remove note" }));
    await waitFor(() => expect(mocks.setDeskMemoryStatus).toHaveBeenCalledWith("mem_1", "Retired"));
  });

  it("offers no change to a note the reader may not edit", async () => {
    mocks.fetchDeskMemories.mockResolvedValue({
      items: [memory({ id: "mem_1", editable: false })],
    });
    renderView();

    const row = within(await screen.findByTestId("memory-mem_1"));
    expect(row.queryByRole("button", { name: "Remove this note" })).toBeNull();
  });

  it("writes a new note about the worker", async () => {
    mocks.fetchDeskMemories.mockResolvedValue({ items: [] });
    mocks.createDeskMemory.mockResolvedValue(memory({ id: "mem_new" }));
    renderView();
    const user = userEvent.setup();

    expect(await screen.findByText(/Agents remember nothing about Maria yet/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Add a note" }));
    await user.type(
      screen.getByRole("textbox", { name: "Something agents should remember about Maria" }),
      "Asks for dock changes by text",
    );
    await user.click(screen.getByRole("button", { name: "Remember" }));
    await waitFor(() =>
      expect(mocks.createDeskMemory).toHaveBeenCalledWith(
        "Asks for dock changes by text",
        { scope: "Organization", roleId: null },
        { kind: "Fact", subject: { type: "Worker", id: "wrk_1" } },
      ),
    );
  });

  it("draws nothing for a reader who may not see agent memories", () => {
    mocks.available = false;
    renderView();

    expect(screen.queryByText("What agents remember about Maria")).toBeNull();
    expect(mocks.fetchDeskMemories).not.toHaveBeenCalled();
  });
});
