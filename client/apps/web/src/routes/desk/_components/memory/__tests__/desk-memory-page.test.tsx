import type {
  DeskMemory,
  DeskMemoryPage as MemoryPage,
  DeskMemorySettings,
} from "@/lib/graphql/desk-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskMemoryPage } from "../desk-memory-page";

const api = vi.hoisted(() => ({
  fetchDeskMemories: vi.fn(),
  fetchDeskMemorySettings: vi.fn(),
  createDeskMemory: vi.fn(),
  reviseDeskMemory: vi.fn(),
  setDeskMemoryStatus: vi.fn(),
  setMemorySavingMode: vi.fn(),
}));

vi.mock("@/lib/graphql/desk-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-memories")>()),
  ...api,
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({ user: { id: "usr_1", name: "Avery Lane", timezone: "UTC" } }),
}));

function memory(overrides: Partial<DeskMemory> & { id: string }): DeskMemory {
  return {
    __typename: "DeskMemory",
    content: "",
    scope: "User",
    roleId: null,
    roleName: "",
    status: "Active",
    source: "User",
    sourceTitle: "",
    useCount: 0,
    lastUsedAt: null,
    createdAt: 1_790_000_000,
    version: 1,
    editable: true,
    ...overrides,
  } as DeskMemory;
}

const own = memory({ id: "m1", content: "Group AR by facility.", createdAt: 1_790_000_300 });
const team = memory({
  id: "m2",
  content: "Acme is billed net-45.",
  scope: "Role",
  roleId: "rol_billing",
  roleName: "Billing",
  source: "Agent",
  sourceTitle: "Acme rate review",
  useCount: 14,
  lastUsedAt: 1_790_000_000,
  editable: false,
  createdAt: 1_790_000_200,
});

const settings: DeskMemorySettings = {
  __typename: "DeskMemorySettings",
  savingMode: "Automatic",
  canShareWithOrganization: false,
  roles: [{ id: "rol_billing", name: "Billing", writable: false }],
} as DeskMemorySettings;

function page(items: DeskMemory[]): MemoryPage {
  return {
    items,
    next: null,
    all: 2,
    counts: [
      { scope: "User", roleId: null, count: 1 },
      { scope: "Role", roleId: "rol_billing", count: 1 },
    ],
  };
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <DeskMemoryPage />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.fetchDeskMemorySettings.mockResolvedValue(settings);
  api.fetchDeskMemories.mockImplementation((filter: { scope: string | null }) =>
    Promise.resolve(
      page(filter.scope === "User" ? [own] : filter.scope === "Role" ? [team] : [own, team]),
    ),
  );
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("DeskMemoryPage", () => {
  it("lists what Desk remembers with where each came from and how it is used", async () => {
    renderPage();

    expect(await screen.findByText("Group AR by facility.")).toBeTruthy();
    expect(screen.getByText("Acme is billed net-45.")).toBeTruthy();
    expect(screen.getByText("Saved Sep 21 from “Acme rate review”")).toBeTruthy();
    expect(screen.getByText("Not used yet")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "What Desk remembers" })).toBeTruthy();
  });

  it("filters by scope with the role's name and counts on the chips", async () => {
    renderPage();
    await screen.findByText("Group AR by facility.");

    const chip = await screen.findByRole("tab", { name: /Billing/ });
    expect(chip.textContent).toBe("Billing1");
    await userEvent.click(chip);

    await waitFor(() =>
      expect(api.fetchDeskMemories).toHaveBeenLastCalledWith(
        { scope: "Role", roleId: "rol_billing", query: "" },
        null,
        50,
        expect.anything(),
      ),
    );
    await waitFor(() => expect(screen.queryByText("Group AR by facility.")).toBeNull());
  });

  it("edits a memory in place and saves the new words", async () => {
    api.reviseDeskMemory.mockResolvedValue({
      ...own,
      content: "Group AR and detention by facility.",
    });
    renderPage();
    await screen.findByText("Group AR by facility.");

    await userEvent.click(screen.getByRole("button", { name: "Edit" }));
    const field = screen.getByRole("textbox", { name: "Memory" });
    await userEvent.clear(field);
    await userEvent.type(field, "Group AR and detention by facility.");
    // The add form has its own Save; the row's is the last.
    await userEvent.click(screen.getAllByRole("button", { name: "Save" }).at(-1)!);

    await waitFor(() =>
      expect(api.reviseDeskMemory).toHaveBeenCalledWith("m1", {
        content: "Group AR and detention by facility.",
        audience: undefined,
        version: 1,
      }),
    );
    await waitFor(() => expect(screen.queryByRole("textbox", { name: "Memory" })).toBeNull());
  });

  it("forgets a memory, says so in its place, and Undo brings it back", async () => {
    api.setDeskMemoryStatus.mockImplementation((id: string, status: string) =>
      Promise.resolve({ ...own, id, status }),
    );
    renderPage();
    await screen.findByText("Group AR by facility.");

    // Once forgotten the server no longer lists it.
    api.fetchDeskMemories.mockResolvedValue(page([team]));
    await userEvent.click(screen.getByRole("button", { name: "Forget" }));

    expect(await screen.findByText("Forgotten. Agents won't use this again.")).toBeTruthy();
    expect(api.setDeskMemoryStatus).toHaveBeenCalledWith("m1", "Retired");

    api.fetchDeskMemories.mockResolvedValue(page([own, team]));
    await userEvent.click(screen.getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(api.setDeskMemoryStatus).toHaveBeenLastCalledWith("m1", "Active"));
    expect(await screen.findByText("Group AR by facility.")).toBeTruthy();
    expect(screen.queryByText("Forgotten. Agents won't use this again.")).toBeNull();
  });

  it("offers no changes on a memory the person may not change", async () => {
    api.fetchDeskMemories.mockResolvedValue(page([team]));
    renderPage();
    await screen.findByText("Acme is billed net-45.");

    expect(screen.queryByRole("button", { name: "Forget" })).toBeNull();
    const [adding, row] = screen.getAllByRole("combobox", { name: "Who it is for" });
    expect(adding).toHaveProperty("disabled", false);
    expect(row).toHaveProperty("disabled", true);
  });

  it("switches how new memories are saved", async () => {
    api.setMemorySavingMode.mockResolvedValue({ ...settings, savingMode: "AskFirst" });
    renderPage();
    await screen.findByText("Agents save useful facts and tell you in the conversation");

    await userEvent.click(screen.getByRole("radio", { name: "Ask me first" }));

    expect(api.setMemorySavingMode).toHaveBeenCalledWith("AskFirst");
    expect(await screen.findByText("Agents ask before saving anything")).toBeTruthy();
  });

  it("says when nothing matches", async () => {
    api.fetchDeskMemories.mockResolvedValue({ ...page([]), all: 0, counts: [] });
    renderPage();

    expect(await screen.findByText("No memories match.")).toBeTruthy();
  });
});
