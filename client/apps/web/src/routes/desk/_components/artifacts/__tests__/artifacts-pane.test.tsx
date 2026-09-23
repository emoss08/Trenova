import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
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

function renderPane(threadId = "athr_1") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ArtifactsPane threadId={threadId} liveArtifactIds={[]} onClose={() => undefined} />
    </QueryClientProvider>,
  );
}

function documentArtifact(id: string, title: string, createdAt: number) {
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
    pinned: false,
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
   * The row of artifacts is a tab list, and walks like one: the arrows move
   * the selection and wrap at the ends, so a keyboard can read a
   * conversation's output without reaching for the pointer.
   */
  it("opens the newest artifact and walks the tabs with the arrows", async () => {
    listArtifacts.mockResolvedValue({
      results: [
        documentArtifact("aart_old", "Morning brief", 10),
        documentArtifact("aart_new", "Handover notes", 20),
      ],
    });

    renderPane("athr_tabs");

    const newest = await screen.findByRole("tab", { name: /Handover notes/ });
    expect(newest).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("heading", { name: "Handover notes" })).toBeInTheDocument();

    fireEvent.keyDown(newest, { key: "ArrowRight" });

    expect(screen.getByRole("tab", { name: /Morning brief/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(await screen.findByRole("heading", { name: "Morning brief" })).toBeInTheDocument();
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
});
