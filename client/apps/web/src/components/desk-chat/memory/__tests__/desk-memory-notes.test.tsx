import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ThreadHistory } from "@/components/assistant/thread-history";
import { queries } from "@/lib/queries";
import type { AssistantMessage, MemoryNote } from "@/types/assistant";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskMemorySaved, initialSaveState, type MemoryCard } from "../desk-memory-notes";

const api = vi.hoisted(() => ({
  fetchDeskMemoriesByIds: vi.fn(),
  fetchDeskMemorySettings: vi.fn(),
  confirmDeskMemory: vi.fn(),
  dismissDeskMemory: vi.fn(),
  reviseDeskMemory: vi.fn(),
  setDeskMemoryStatus: vi.fn(),
}));

vi.mock("@/lib/graphql/desk-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-memories")>()),
  ...api,
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({ user: { id: "usr_1", name: "Avery Lane", timezone: "UTC" } }),
}));

function card(overrides: Partial<MemoryCard>): MemoryCard {
  return {
    id: "amem_1",
    content: "Read the move before assigning it.",
    kind: "Procedure",
    scope: "Agent",
    roleId: null,
    roleName: null,
    status: "Active",
    source: "Reflection",
    sourceTitle: "Night loads",
    createdAt: 1_790_000_000,
    version: 1,
    editable: true,
    reason: null,
    replaces: null,
    replacedBy: null,
    ...overrides,
  };
}

function renderCard(
  memory: MemoryCard,
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } }),
) {
  return render(
    <QueryClientProvider client={client}>
      <DeskMemorySaved card={memory} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  for (const mock of Object.values(api)) {
    mock.mockReset();
  }
  api.fetchDeskMemorySettings.mockResolvedValue({
    savingMode: "Automatic",
    canShareWithOrganization: false,
    roles: [],
  });
});

/*
A memory is retired when a person forgets it and when a newer one replaces it.
Only the first may be brought back from the card: bringing back a replaced
memory would set it beside the one that took its place.
*/
describe("initialSaveState", () => {
  it("reads a forgotten memory as removed and a replaced one as replaced", () => {
    expect(initialSaveState("Retired")).toBe("removed");
    expect(initialSaveState("Retired", false)).toBe("removed");
    expect(initialSaveState("Retired", true)).toBe("replaced");
  });

  it("leaves every other status as it was", () => {
    expect(initialSaveState("Active", true)).toBe("saved");
    expect(initialSaveState("Paused")).toBe("saved");
    expect(initialSaveState("Suggested", true)).toBe("ask");
    expect(initialSaveState("Dismissed")).toBe("declined");
  });
});

const STEPS =
  "1. If `search_shipments` finds nothing with a status filter, keep looking. 2. Call `list_shipments` with no status filter.";

describe("DeskMemorySaved", () => {
  it("names the memory that replaced a retired one, and offers no undo", () => {
    renderCard(
      card({
        status: "Retired",
        replacedBy: {
          id: "amem_2",
          content: "Read the move and its stops before assigning it.",
          status: "Active",
        },
      }),
    );

    expect(screen.getByText("Replaced by a newer memory")).toBeInTheDocument();
    expect(screen.getByText("Read the move and its stops before assigning it.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Undo" })).not.toBeInTheDocument();
  });

  it("brings back a memory the person forgot", async () => {
    api.setDeskMemoryStatus.mockResolvedValue(card({ status: "Active", version: 2 }));
    renderCard(card({ status: "Retired" }));

    expect(screen.getByText("Removed from memory")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(api.setDeskMemoryStatus).toHaveBeenCalledWith("amem_1", "Active"));
    expect(await screen.findByText("Learned steps")).toBeInTheDocument();
  });

  it("asks in one sentence naming who the steps are for, and lists them with their code", () => {
    renderCard(card({ status: "Suggested", content: STEPS }));

    expect(screen.getByRole("button", { name: "Visible to everyone" })).toHaveTextContent(
      "everyone",
    );
    expect(screen.getByText(/Keep these steps for/)).toBeInTheDocument();
    expect(screen.getByText("search_shipments").tagName).toBe("CODE");
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByRole("button", { name: "Not now" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep" })).toBeInTheDocument();
  });

  it("asks about a fact, and about a memory the person asked to be asked about, in their own words", () => {
    const { unmount } = renderCard(card({ status: "Suggested", kind: "Fact" }));
    expect(screen.getByText(/Keep this for/)).toBeInTheDocument();
    unmount();

    renderCard(card({ status: "Suggested", kind: "Instruction", source: "Agent", scope: "User" }));
    expect(screen.getByText(/Remember this for/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Visible to just you" })).toBeInTheDocument();
  });

  it("chooses who the steps are for from the word in the sentence, and keeps them for that team", async () => {
    api.fetchDeskMemorySettings.mockResolvedValue({
      savingMode: "AskFirst",
      canShareWithOrganization: false,
      roles: [{ id: "rol_1", name: "Dispatch", writable: true }],
    });
    api.confirmDeskMemory.mockResolvedValue(
      card({ status: "Active", scope: "Role", roleId: "rol_1", roleName: "Dispatch", version: 2 }),
    );
    renderCard(card({ status: "Suggested", content: STEPS }));

    fireEvent.click(screen.getByRole("button", { name: "Visible to everyone" }));
    const choice = await screen.findByRole("radiogroup", { name: "Visible to" });
    expect(within(choice).getByRole("radio", { name: /Everyone/ })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    fireEvent.click(await within(choice).findByRole("radio", { name: /Dispatch/ }));

    expect(screen.getByRole("button", { name: "Visible to Dispatch" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Keep" }));

    await waitFor(() =>
      expect(api.confirmDeskMemory).toHaveBeenCalledWith(
        "amem_1",
        STEPS,
        { scope: "Role", roleId: "rol_1" },
        1,
      ),
    );
    expect(await screen.findByText("Kept for Dispatch")).toBeInTheDocument();
    expect(screen.getByText(/Desk will follow these next time/)).toBeInTheDocument();
  });

  it("folds a kept memory to a line with Undo, which takes it back out", async () => {
    const offered = card({ status: "Suggested" });
    api.confirmDeskMemory.mockResolvedValue({ ...offered, status: "Active", version: 2 });
    api.setDeskMemoryStatus.mockResolvedValue({ ...offered, status: "Retired", version: 3 });
    renderCard(offered);

    fireEvent.click(screen.getByRole("button", { name: "Keep" }));
    expect(await screen.findByText("Kept for everyone")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Keep" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Undo" }));
    await waitFor(() => expect(api.setDeskMemoryStatus).toHaveBeenCalledWith("amem_1", "Retired"));
    expect(await screen.findByText("Removed from memory")).toBeInTheDocument();
  });

  it("turns an offer down with Not now, and Undo asks again", async () => {
    const offered = card({ status: "Suggested" });
    api.dismissDeskMemory.mockResolvedValue({ ...offered, status: "Dismissed", version: 2 });
    renderCard(offered);

    fireEvent.click(screen.getByRole("button", { name: "Not now" }));
    expect(await screen.findByText("Not kept")).toBeInTheDocument();
    expect(api.dismissDeskMemory).toHaveBeenCalledWith("amem_1");

    fireEvent.click(screen.getByRole("button", { name: "Undo" }));
    expect(screen.getByRole("button", { name: "Keep" })).toBeInTheDocument();
  });

  it("edits the steps where they stand: one per line, Enter keeps the edit, Esc drops it", async () => {
    api.confirmDeskMemory.mockImplementation(async (_id: string, content: string) =>
      card({ status: "Active", content, version: 2 }),
    );
    renderCard(card({ status: "Suggested", content: STEPS }));

    fireEvent.click(screen.getByRole("button", { name: "Edit memory" }));
    let editor = screen.getByRole("textbox", { name: "Memory" });
    expect(editor).toHaveFocus();
    expect(editor).toHaveValue(
      "If `search_shipments` finds nothing with a status filter, keep looking.\nCall `list_shipments` with no status filter.",
    );
    fireEvent.change(editor, { target: { value: "Something else" } });
    fireEvent.keyDown(editor, { key: "Escape" });
    expect(screen.queryByRole("textbox", { name: "Memory" })).not.toBeInTheDocument();
    expect(screen.getByText("search_shipments")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit memory" }));
    editor = screen.getByRole("textbox", { name: "Memory" });
    fireEvent.change(editor, { target: { value: "Search without a status\nMatch on the PRO" } });
    fireEvent.keyDown(editor, { key: "Enter" });
    expect(screen.queryByRole("textbox", { name: "Memory" })).not.toBeInTheDocument();
    expect(screen.getByText("Match on the PRO")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Keep" }));
    await waitFor(() =>
      expect(api.confirmDeskMemory).toHaveBeenCalledWith(
        "amem_1",
        "1. Search without a status 2. Match on the PRO",
        { scope: "Agent", roleId: null },
        1,
      ),
    );
  });

  it("edits a saved memory from its Edit button and saves it with Enter", async () => {
    api.reviseDeskMemory.mockResolvedValue(card({ content: "Read the stops first.", version: 2 }));
    renderCard(card({ kind: "Instruction" }));

    fireEvent.click(screen.getByRole("button", { name: "Edit memory" }));
    const editor = screen.getByRole("textbox", { name: "Memory" });
    fireEvent.change(editor, { target: { value: "Read the stops first." } });
    fireEvent.keyDown(editor, { key: "Enter" });

    await waitFor(() =>
      expect(api.reviseDeskMemory).toHaveBeenCalledWith("amem_1", {
        content: "Read the stops first.",
        audience: { scope: "Agent", roleId: null },
        version: 1,
      }),
    );
    expect(await screen.findByText("Read the stops first.")).toBeInTheDocument();
  });

  it("shows a saved lesson with who it is for, and opens why it was kept and what it replaced", () => {
    renderCard(
      card({
        reason: "Assigning failed until the move was read.",
        replaces: { id: "amem_0", content: "1. Assign. 2. Confirm.", status: "Retired" },
      }),
    );

    expect(screen.getByText("Learned steps")).toBeInTheDocument();
    expect(screen.getByText(/^Everyone · /)).toBeInTheDocument();
    expect(screen.getByText("Replaces earlier steps")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit memory" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Undo" })).toBeInTheDocument();

    const why = screen.getByRole("button", { name: "Why" });
    expect(why).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Assigning failed until the move was read.").closest("[inert]")).not.toBeNull();

    fireEvent.click(why);

    expect(why).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Assigning failed until the move was read.").closest("[inert]")).toBeNull();
    expect(screen.getByText("Before")).toBeInTheDocument();
    expect(screen.getByText("Assign.")).toBeInTheDocument();
  });

  it("keeps long steps inside the scroll area", () => {
    const long = Array.from({ length: 12 }, (_, index) => `${index + 1}. Step ${index + 1}.`).join(" ");
    const { container } = renderCard(card({ content: long }));

    const area = container.querySelector('[data-slot="scroll-area"]');
    expect(area).not.toBeNull();
    expect(area?.querySelectorAll("li")).toHaveLength(12);
  });

  it("keeps the plain wording for a memory the person asked to save, with no Why", () => {
    renderCard(card({ source: "Agent", kind: "Instruction", scope: "User" }));

    expect(screen.getByText("Saved to memory")).toBeInTheDocument();
    expect(screen.getByText(/^Just you · /)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Why" })).not.toBeInTheDocument();
  });

  it("keeps the saved state in the thread's history, so the card reads the same when reopened", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const offered = card({ status: "Suggested" });
    const note: MemoryNote = offered;
    const key = queries.assistant.messages("thr_1").queryKey;
    client.setQueryData<ThreadHistory>(key, {
      pages: [
        {
          results: [{ id: "amsg_1", memories: [note] } as unknown as AssistantMessage],
          hasMore: false,
          total: 1,
          limit: 0,
        } as unknown as ThreadHistory["pages"][number],
      ],
      pageParams: [undefined],
    });
    api.confirmDeskMemory.mockResolvedValue({ ...offered, status: "Active", version: 2 });

    const { unmount } = renderCard(offered, client);
    fireEvent.click(screen.getByRole("button", { name: "Keep" }));

    await waitFor(() => expect(screen.getByText("Kept for everyone")).toBeInTheDocument());
    const kept = client.getQueryData<ThreadHistory>(key)?.pages[0].results[0].memories?.[0];
    expect(kept?.status).toBe("Active");
    expect(kept?.version).toBe(2);

    unmount();
    renderCard(kept as MemoryCard, client);
    expect(screen.getByText("Learned steps")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Keep" })).not.toBeInTheDocument();
  });
});
