import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { LiveArtifacts } from "../../desk-layout";
import { ArtifactsPane } from "../artifacts-pane";

const listArtifacts = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      listArtifacts: (...args: unknown[]) => listArtifacts(...args),
      pinArtifact: vi.fn(),
    },
  },
}));

afterEach(() => {
  cleanup();
  listArtifacts.mockReset();
});

const NO_LIVE: LiveArtifacts = { ids: [], revision: 0 };

function renderPane(threadId = "athr_1", live: LiveArtifacts = NO_LIVE) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const pane = (liveArtifacts: LiveArtifacts) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ArtifactsPane
          threadId={threadId}
          liveArtifacts={liveArtifacts}
          onClose={() => undefined}
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(pane(live));

  return { ...view, rerenderWith: (next: LiveArtifacts) => view.rerender(pane(next)) };
}

function documentArtifact(id: string, title: string, createdAt: number, pinned = false) {
  return {
    id,
    threadId: "athr_tabs",
    messageId: null,
    runId: null,
    proposalId: null,
    planId: null,
    kind: "document",
    status: "Ready",
    title,
    payload: { body: "" },
    sourceToolCallId: "",
    pinned,
    createdAt,
    updatedAt: createdAt,
  };
}

describe("the artifacts pane", () => {
  it("names what to ask for, with the question itself rather than a placeholder", async () => {
    listArtifacts.mockResolvedValue({ results: [] });

    renderPane();

    expect(await screen.findByText("Nothing here yet")).toBeInTheDocument();
    expect(screen.getByText("“Which shipments are in transit?”")).toBeInTheDocument();
    expect(screen.queryByText(/\{example\}/)).toBeNull();
  });

  /**
   * A list that could not be read is not an empty one.
   *
   * The pane used to say "Nothing here yet" whatever went wrong, which is the
   * one thing a person who just watched a table being made cannot believe.
   */
  it("says the list could not be read, and reads it again on request", async () => {
    listArtifacts.mockRejectedValueOnce(new Error("offline"));
    listArtifacts.mockResolvedValueOnce({ results: [] });

    renderPane();

    expect(
      await screen.findByText("This conversation's artifacts could not be loaded."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Nothing here yet")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Nothing here yet")).toBeInTheDocument();
  });

  /**
   * The header is a document switcher, and walks like one: the arrows move
   * to the next and previous artifact and wrap at the ends, so a keyboard can
   * read a conversation's output without reaching for the pointer.
   */
  it("opens the newest artifact and walks the list with the arrows", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        documentArtifact("aart_old", "Morning brief", 10),
        documentArtifact("aart_new", "Handover notes", 20),
      ],
    });

    renderPane("athr_tabs");

    const switcher = await screen.findByRole("button", { name: "Handover notes" });
    expect(screen.getByRole("heading", { name: "Handover notes" })).toBeInTheDocument();
    expect(screen.getByText("1 of 2")).toBeInTheDocument();

    fireEvent.keyDown(switcher, { key: "ArrowRight" });

    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();
    expect(screen.getByText("2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Open artifact" })).toHaveTextContent(
      "Morning brief, 2 of 2",
    );

    fireEvent.keyDown(screen.getByRole("button", { name: "Morning brief" }), {
      key: "ArrowRight",
    });
    expect(await screen.findByRole("heading", { name: "Handover notes" })).toBeInTheDocument();

    fireEvent.keyDown(screen.getByRole("button", { name: "Handover notes" }), {
      key: "ArrowLeft",
    });
    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();
  });

  it("moves with the previous and next buttons", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        documentArtifact("aart_old", "Morning brief", 10),
        documentArtifact("aart_new", "Handover notes", 20),
      ],
    });

    renderPane("athr_buttons");
    await screen.findByRole("heading", { name: "Handover notes" });

    await userEvent.click(screen.getByRole("button", { name: "Next artifact" }));
    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Previous artifact" }));
    expect(await screen.findByRole("heading", { name: "Handover notes" })).toBeInTheDocument();
  });

  /**
   * The switcher lists everything the conversation made, newest first with
   * what the person pinned above it, so twelve artifacts never become a row
   * of pills that runs off the pane.
   */
  it("lists the artifacts newest first with pinned ones first, searchable", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        documentArtifact("aart_old", "Morning brief", 10),
        documentArtifact("aart_pinned", "Late loads", 5, true),
        documentArtifact("aart_new", "Handover notes", 20),
      ],
    });

    renderPane("athr_list");

    await userEvent.click(await screen.findByRole("button", { name: "Handover notes" }));

    const list = await screen.findByRole("listbox", { name: "Artifacts" });
    const options = within(list).getAllByRole("option");
    expect(options.map((option) => option.textContent)).toEqual([
      expect.stringContaining("Late loads"),
      expect.stringContaining("Handover notes"),
      expect.stringContaining("Morning brief"),
    ]);
    expect(within(list).getByRole("option", { name: /Handover notes/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    await userEvent.type(screen.getByRole("combobox", { name: "Search artifacts" }), "morn");
    expect(within(list).getAllByRole("option")).toHaveLength(1);

    await userEvent.click(within(list).getByRole("option", { name: /Morning brief/ }));
    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
  });

  it("re-reads the list and opens the newest when a turn produces something", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        documentArtifact("aart_old", "Morning brief", 10),
        documentArtifact("aart_new", "Handover notes", 20),
      ],
    });

    const pane = renderPane("athr_live");
    await screen.findByRole("heading", { name: "Handover notes" });
    expect(listArtifacts).toHaveBeenCalledTimes(1);

    pane.rerenderWith({ ids: ["aart_old"], revision: 1 });

    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();
    await waitFor(() => expect(listArtifacts).toHaveBeenCalledTimes(2));
  });

  it("says where an artifact came from beneath its title", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        {
          ...documentArtifact("aart_run", "Detention last week", 10),
          kind: "report_run",
          payload: {},
        },
      ],
    });

    renderPane("athr_provenance");

    expect(await screen.findByText("Report · from Report builder")).toBeInTheDocument();
    expect(screen.getByText("This run has no id to follow.")).toBeInTheDocument();
  });

  // A table the server read together from several calls says so, because
  // "Billing queue item (11)" alone reads as one search's result.
  it("says when a table was read together from several calls", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        {
          ...documentArtifact("aart_bunch", "Billing queue item (11)", 10),
          kind: "table_view",
          payload: {
            display: 1,
            tool: "get_billing_queue_item",
            entity: "billing_queue_item",
            columns: [{ key: "name", label: "Name", type: "text" }],
            rows: [{ id: "bqi_1", name: "Peak Distributing" }],
            rowCount: 1,
            searchedFor: [],
            bunched: true,
            calls: Array.from({ length: 11 }, (_, index) => `call_${index}`),
          },
        },
      ],
    });

    renderPane("athr_bunched");

    expect(await screen.findByText("Read together from 11 calls")).toBeInTheDocument();
  });
});
