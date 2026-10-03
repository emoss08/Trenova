import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { useDeskStore } from "@/stores/desk-store";
import { describe, expect, it, vi } from "vitest";
import { groupLineages, lineageContaining } from "../desk-lineage";
import { changedCells, gridOf } from "../desk-table-body";
import { DeskWorkspace } from "../desk-workspace";

const state = vi.hoisted(() => ({ artifacts: [] as unknown[] }));

vi.mock("@/lib/queries", () => ({
  queries: {
    assistant: {
      artifacts: (threadId: string) => ({
        queryKey: ["assistant-artifacts", threadId],
        queryFn: () => page(),
      }),
      artifactLineage: (threadId: string, id: string) => ({
        queryKey: ["assistant-artifacts", threadId, "lineage", id],
        queryFn: () => ({ results: [] }),
      }),
      proposals: (threadId: string) => ({
        queryKey: ["assistant", "proposals", threadId],
        queryFn: () => ({ results: [] }),
      }),
    },
  },
}));

function page(needle = "") {
  const results = (state.artifacts as AssistantArtifact[]).filter((artifact) =>
    artifact.title.toLowerCase().includes(needle.toLowerCase()),
  );
  return {
    results,
    total: results.length,
    nextCursor: "",
    counts: { all: results.length, pinned: 0, families: { table: results.length } },
  };
}

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      listArtifacts: (_threadId: string, options: { q?: string }) =>
        Promise.resolve(page(options.q ?? "")),
    },
  },
}));

let clock = Math.floor(Date.now() / 1000) - 3600;

function artifact(overrides: Partial<AssistantArtifact> & { id: string }): AssistantArtifact {
  clock += 60;
  return {
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "Workers",
    payload: {},
    sourceToolCallId: "",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: overrides.id,
    turn: "",
    createdAt: clock,
    updatedAt: clock,
    ...overrides,
  };
}

function workers(id: string, status: string, overrides: Partial<AssistantArtifact> = {}) {
  return artifact({
    id,
    payload: {
      display: 1,
      tool: "list_workers",
      entity: "workers",
      recordEntity: "worker",
      columns: [
        { key: "name", label: "Name", type: "text" },
        { key: "status", label: "Status", type: "status" },
      ],
      rows: [
        { id: "wrk_1", name: "Avery Lane", status },
        { id: "wrk_2", name: "Dana Ortiz", status: "Active" },
      ],
      rowCount: 2,
    },
    ...overrides,
  });
}

describe("groupLineages", () => {
  it("folds later reads into the first artifact's lineage, pinned first then newest", () => {
    const first = workers("art_1", "Active");
    const card = artifact({ id: "art_2", kind: "entity_card", title: "SEED-SHP-003" });
    const second = workers("art_3", "Inactive", { lineageId: "art_1", lineageSeq: 2 });
    const pinned = artifact({
      id: "art_0",
      kind: "document",
      title: "Brief",
      pinned: true,
      createdAt: 1,
    });

    const lineages = groupLineages([first, card, second, pinned]);

    expect(lineages.map((lineage) => lineage.id)).toEqual(["art_0", "art_1", "art_2"]);
    expect(lineages[1].versions.map((version) => version.id)).toEqual(["art_1", "art_3"]);
    expect(lineages[1].latest.id).toBe("art_3");
    expect(lineageContaining(lineages, "art_3")?.id).toBe("art_1");
  });
});

describe("changedCells", () => {
  it("marks the cells that differ from the version before, row by record", () => {
    const before = gridOf(workers("art_1", "Active"));
    const after = gridOf(workers("art_3", "Inactive"));

    const changed = changedCells(after, before);

    expect(changed.size).toBe(1);
    expect([...changed][0]).toMatch(/:status$/);
    expect(changedCells(after, null).size).toBe(0);
  });
});

function renderWorkspace(onClose = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskWorkspace
          threadId="athr_1"
          liveArtifacts={{ ids: [], revision: 0 }}
          onClose={onClose}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onClose };
}

describe("DeskWorkspace", () => {
  it("says what will land here before anything has", async () => {
    state.artifacts = [];
    const { onClose } = renderWorkspace();

    expect(await screen.findByText("No artifacts yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Hide artifacts" }));
    expect(onClose).toHaveBeenCalled();
  });

  it("opens the latest version and goes back to an earlier one", async () => {
    state.artifacts = [
      workers("art_1", "Active"),
      workers("art_3", "Inactive", { lineageId: "art_1", lineageSeq: 2 }),
    ];
    renderWorkspace();

    expect(await screen.findByText("Inactive")).toBeInTheDocument();
    expect(screen.getByText("1 changed")).toBeInTheDocument();

    fireEvent.click(screen.getByTitle("Versions"));
    fireEvent.click(within(screen.getByRole("listbox")).getByRole("option", { name: /v1/ }));

    expect(await screen.findAllByText("Active")).not.toHaveLength(0);
    expect(screen.queryByText("Inactive")).not.toBeInTheDocument();
  });

  it("searches everything the conversation made, on the server", async () => {
    state.artifacts = [
      workers("art_1", "Active"),
      artifact({
        id: "art_2",
        kind: "document",
        title: "Storm brief",
        payload: { body: "Two loads." },
      }),
    ];
    renderWorkspace();
    await screen.findByText("Storm brief", { selector: "h2" });

    act(() => useDeskStore.getState().setBrowsing(true));
    const search = await screen.findByRole("textbox", { name: "Search artifacts" });
    fireEvent.change(search, { target: { value: "work" } });

    const titles = () =>
      [...document.querySelectorAll(".dk-axb-rt b")].map((title) => title.textContent);
    await waitFor(() => expect(titles()).toEqual(["Workers"]));
    fireEvent.keyDown(search, { key: "Enter" });
    expect(await screen.findByText("Avery Lane")).toBeInTheDocument();
  });

  it("fans the stack on a click and folds it on Esc without closing the pane", async () => {
    state.artifacts = [workers("art_1", "Active"), artifact({ id: "art_2", title: "Rates" })];
    act(() => useDeskStore.getState().setBrowsing(false));
    const { onClose } = renderWorkspace();
    await screen.findByText("Avery Lane");

    const front = screen.getByTitle("Switch artifact");
    expect(document.querySelector(".dk-ax-stack.dk-fan")).toBeNull();
    fireEvent.click(front);
    expect(document.querySelector(".dk-ax-stack.dk-fan")).not.toBeNull();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(document.querySelector(".dk-ax-stack.dk-fan")).toBeNull();
    expect(onClose).not.toHaveBeenCalled();
  });
});
