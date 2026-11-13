import type { DeskMemory } from "@/lib/graphql/desk-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RecordMemories } from "../record-memories";

const api = vi.hoisted(() => ({
  fetchDeskMemories: vi.fn(),
  createDeskMemory: vi.fn(),
  reviseDeskMemory: vi.fn(),
  setDeskMemoryStatus: vi.fn(),
}));
const access = vi.hoisted(() => ({ allowed: true }));

vi.mock("@/lib/graphql/desk-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-memories")>()),
  ...api,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: access.allowed }),
}));

const SUBJECT = { type: "Customer", id: "cus_1" } as const;

function memory(overrides: Partial<DeskMemory> = {}): DeskMemory {
  return {
    id: "amem_1",
    content: "The receiving dock closes at 2pm on Fridays.",
    kind: "Fact",
    scope: "Organization",
    roleId: null,
    roleName: "",
    subjectType: "Customer",
    subjectLabel: "Acme Manufacturing",
    status: "Active",
    source: "User",
    sourceTitle: "",
    useCount: 0,
    lastUsedAt: null,
    createdAt: 1_800_000_000,
    version: 2,
    editable: true,
    reason: "",
    quotes: [],
    replaces: null,
    replacedBy: null,
    ...overrides,
  } as DeskMemory;
}

function page(items: DeskMemory[]) {
  return { items, next: null, all: items.length, counts: [] };
}

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <RecordMemories subject={SUBJECT} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  access.allowed = true;
  api.fetchDeskMemories.mockResolvedValue(page([]));
  api.createDeskMemory.mockResolvedValue(memory());
  api.reviseDeskMemory.mockResolvedValue(memory());
  api.setDeskMemoryStatus.mockResolvedValue(memory({ status: "Retired" }));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RecordMemories", () => {
  it("renders nothing and reads nothing for someone who may not use the agents", () => {
    access.allowed = false;
    renderPanel();

    expect(screen.queryByRole("region", { name: "What agents remember" })).not.toBeInTheDocument();
    expect(api.fetchDeskMemories).not.toHaveBeenCalled();
  });

  it("reads only the memories about this record", async () => {
    renderPanel();

    await waitFor(() => expect(api.fetchDeskMemories).toHaveBeenCalled());
    expect(api.fetchDeskMemories.mock.calls[0]?.[0]).toEqual({
      scope: null,
      roleId: null,
      query: "",
      subject: SUBJECT,
    });
    expect(await screen.findByText("Nothing is remembered about it yet.")).toBeInTheDocument();
  });

  it("lists what is kept, with who it is for and whether it waits for approval", async () => {
    api.fetchDeskMemories.mockResolvedValue(
      page([
        memory(),
        memory({
          id: "amem_2",
          kind: "Instruction",
          content: "Call an hour before every delivery.",
          status: "Suggested",
          editable: false,
        }),
      ]),
    );
    renderPanel();

    const rows = within(await screen.findByRole("list")).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText("Fact")).toBeInTheDocument();
    expect(within(rows[0]).getByText("Organization · Not used yet")).toBeInTheDocument();
    expect(within(rows[0]).getByRole("button", { name: "Edit" })).toBeInTheDocument();
    expect(within(rows[1]).getByText("Rule")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Waiting for an administrator")).toBeInTheDocument();
    expect(within(rows[1]).queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(within(rows[1]).queryByRole("button", { name: "Forget" })).not.toBeInTheDocument();
  });

  it("keeps a new fact under the record, for everyone unless told otherwise", async () => {
    renderPanel();

    const box = await screen.findByRole("textbox", {
      name: "Something agents should remember about this record",
    });
    const remember = screen.getByRole("button", { name: "Remember" });
    expect(remember).toBeDisabled();

    await userEvent.type(box, "  Call an hour before every delivery.  ");
    await userEvent.click(remember);

    expect(api.createDeskMemory).toHaveBeenCalledWith(
      "Call an hour before every delivery.",
      { scope: "Organization", roleId: null },
      { kind: "Fact", subject: SUBJECT },
    );
    await waitFor(() => expect(box).toHaveValue(""));
    expect(api.fetchDeskMemories).toHaveBeenCalledTimes(2);
  });

  it("keeps a rule for the person alone when they say so", async () => {
    renderPanel();

    await userEvent.type(
      await screen.findByRole("textbox", {
        name: "Something agents should remember about this record",
      }),
      "Quote in Canadian dollars.",
    );
    await userEvent.click(screen.getByRole("radio", { name: "Rule" }));
    await userEvent.click(screen.getByRole("radio", { name: "Just me" }));
    await userEvent.click(screen.getByRole("button", { name: "Remember" }));

    expect(api.createDeskMemory).toHaveBeenCalledWith(
      "Quote in Canadian dollars.",
      { scope: "User", roleId: null },
      { kind: "Instruction", subject: SUBJECT },
    );
  });

  it("changes what a memory says at the version read", async () => {
    api.fetchDeskMemories.mockResolvedValue(page([memory()]));
    renderPanel();

    await userEvent.click(await screen.findByRole("button", { name: "Edit" }));
    const box = screen.getByRole("textbox", { name: "What agents should remember" });
    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeDisabled();

    await userEvent.clear(box);
    await userEvent.type(box, "The receiving dock closes at 3pm on Fridays.");
    await userEvent.click(save);

    expect(api.reviseDeskMemory).toHaveBeenCalledWith("amem_1", {
      content: "The receiving dock closes at 3pm on Fridays.",
      version: 2,
    });
  });

  it("forgets a memory that no longer holds", async () => {
    api.fetchDeskMemories.mockResolvedValue(page([memory()]));
    renderPanel();

    await userEvent.click(await screen.findByRole("button", { name: "Forget" }));

    expect(api.setDeskMemoryStatus).toHaveBeenCalledWith("amem_1", "Retired");
    await waitFor(() => expect(api.fetchDeskMemories).toHaveBeenCalledTimes(2));
  });

  it("says so when the memories cannot be read", async () => {
    api.fetchDeskMemories.mockRejectedValue(new Error("offline"));
    renderPanel();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The memories could not be read. Try again in a moment.",
    );
  });
});
