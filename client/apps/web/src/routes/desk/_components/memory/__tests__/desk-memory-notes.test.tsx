import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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

function renderCard(memory: MemoryCard) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

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
    expect(
      screen.getByText("“Read the move and its stops before assigning it.”"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Undo" })).not.toBeInTheDocument();
  });

  it("keeps Undo for a memory the person forgot", () => {
    renderCard(card({ status: "Retired" }));

    expect(screen.getByText("Removed from memory")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Undo" })).toBeInTheDocument();
  });

  it("asks before keeping steps an agent learned, and says why and what they replace", () => {
    renderCard(
      card({
        status: "Suggested",
        reason: "Assigning failed until the move was read.",
        replaces: { id: "amem_0", content: "Assign the move straight away.", status: "Active" },
      }),
    );

    expect(
      screen.getByText("Learned the steps that worked. Keep them for next time?"),
    ).toBeInTheDocument();
    expect(screen.getByText("Assigning failed until the move was read.")).toBeInTheDocument();
    expect(screen.getByText("“Assign the move straight away.”")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save memory" })).toBeInTheDocument();
  });

  it("asks before keeping a fact an agent learned in different words", () => {
    renderCard(card({ status: "Suggested", kind: "Fact" }));

    expect(
      screen.getByText("Learned something from this conversation. Keep it?"),
    ).toBeInTheDocument();
  });

  it("marks a kept lesson as learned and shows why it was kept", () => {
    renderCard(card({ reason: "Assigning failed until the move was read." }));

    expect(screen.getByText("Learned the steps that worked")).toBeInTheDocument();
    expect(screen.getByText("Assigning failed until the move was read.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Undo" })).toBeInTheDocument();
  });

  it("keeps the plain wording for a memory the person asked to save", () => {
    renderCard(card({ source: "Agent", kind: "Instruction", scope: "User" }));

    expect(screen.getByText("Saved to memory")).toBeInTheDocument();
    expect(screen.queryByText("Why")).not.toBeInTheDocument();
  });
});
