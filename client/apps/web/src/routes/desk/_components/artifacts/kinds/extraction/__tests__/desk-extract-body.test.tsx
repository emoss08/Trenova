import { assistant } from "@/lib/queries/assistant";
import { documentPageImageEndpoint } from "@/services/document";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantProposal } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskExtractBody } from "../desk-extract-body";
import { DOCUMENT, THREAD, artifactOf, currentPayload, legacyPayload } from "./extraction-fixtures";

const mocks = vi.hoisted(() => ({ download: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});

type Setup = {
  payload?: Record<string, unknown>;
  proposals?: Partial<AssistantProposal>[];
  replying?: boolean;
  decisions?: { id: string; kind: string; proposalId: string }[];
  onOpenArtifact?: (id: string) => void;
};

function renderBody({
  payload = currentPayload(),
  proposals = [],
  replying = false,
  decisions = [],
  onOpenArtifact,
}: Setup = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  client.setQueryData(assistant.proposals(THREAD).queryKey, {
    results: proposals,
    count: proposals.length,
  });
  client.setQueryData(assistant.activeTurns().queryKey, {
    items: replying
      ? [{ turnId: "atrn_1", threadId: THREAD, threadTitle: "", origin: "Desk", startedAt: 1 }]
      : [],
  });
  client.setQueryData(assistant.artifacts(THREAD)._ctx.summary.queryKey, {
    results: decisions,
    total: decisions.length,
    nextCursor: "",
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DeskExtractBody artifact={artifactOf(payload)} onOpenArtifact={onOpenArtifact} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function option(name: RegExp) {
  return screen.getByRole("option", { name });
}

async function drawnPage(page: number) {
  const image = await screen.findByAltText(`Page ${page} of Northline_RateCon.pdf`);
  fireEvent.load(image);
  return image;
}

beforeEach(() => {
  mocks.download.mockImplementation(async (endpoint: string) => ({
    blob: new Blob([endpoint], { type: "image/webp" }),
    fileName: "page.webp",
  }));
  let made = 0;
  vi.spyOn(URL, "createObjectURL").mockImplementation(() => {
    made += 1;
    return `blob:page/${made}`;
  });
  vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  useDeskStore.setState({ asks: {}, documentAsks: {} });
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  mocks.download.mockReset();
});

describe("DeskExtractBody", () => {
  it("fetches the page through the app's request helper and boxes each value once it is drawn", async () => {
    const { container } = renderBody();

    await waitFor(() =>
      expect(mocks.download).toHaveBeenCalledWith(
        documentPageImageEndpoint(DOCUMENT, 1),
        "page.webp",
        { signal: expect.any(AbortSignal) },
      ),
    );
    expect(container.querySelectorAll(".dk-ex-box")).toHaveLength(0);

    await drawnPage(1);

    const load = container.querySelector<HTMLElement>('.dk-ex-box[title="Load #: NL-482913"]');
    expect(load).not.toBeNull();
    expect(load?.style.left).toBe("70.75%");
    expect(load?.style.top).toBe("8.5%");
    expect(load?.style.width).toBe("11.25%");
    expect(load?.style.height).toBe("1.4%");
  });

  it("counts what to check, what the document lacked and what was read", () => {
    renderBody();

    expect(screen.getByRole("button", { name: "4 to check" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "3 not on the document" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "7 read" })).toBeInTheDocument();
  });

  it("says why a field needs a look, and where a missing one would have been", () => {
    renderBody();

    expect(
      option(/^Bill to: Northline Logistics, needs a look, 71 percent confident, page 1$/),
    ).toHaveTextContent("Marked for review · 71%");
    expect(option(/^Delivery window:/)).toHaveTextContent(
      "Two readings disagree10/21/202610/22/2026",
    );
    expect(option(/^Pro #: not on the document$/)).toHaveTextContent("Not found");
    expect(option(/^Load #: NL-482913, read with 97 percent confidence/)).not.toHaveTextContent(
      "%",
    );
    expect(option(/^Rate: 2465.00, read with 86 percent confidence/)).toHaveTextContent("86%");
  });

  it("turns to the page a chosen field was read from", async () => {
    renderBody();
    await drawnPage(1);

    fireEvent.click(option(/^Name: Anyco Clothes #612/));

    await drawnPage(2);
    expect(screen.getByRole("button", { name: "Page 2, with fields to check" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(option(/^Name: Anyco Clothes #612/)).toHaveAttribute("aria-selected", "true");
  });

  it("chooses a field when its box is pressed, and lights the box when its field is chosen", async () => {
    const { container } = renderBody();
    await drawnPage(1);

    const box = container.querySelector<HTMLElement>(
      '.dk-ex-box[title="Bill to: Northline Logistics"]',
    );
    fireEvent.click(box as HTMLElement);

    expect(option(/^Bill to:/)).toHaveAttribute("aria-selected", "true");
    expect(box).toHaveClass("dk-on");
    expect(container.querySelector(".dk-ex-flag")).toHaveTextContent("Bill to");

    fireEvent.mouseEnter(option(/^Load #:/));
    expect(container.querySelector('.dk-ex-box[title="Load #: NL-482913"]')).toHaveClass("dk-hot");
  });

  it("walks the list with the arrow keys and lets go with Escape", async () => {
    renderBody();
    const list = screen.getByRole("listbox", { name: "What was read" });

    fireEvent.keyDown(list, { key: "ArrowDown" });
    expect(option(/^Shipper:/)).toHaveAttribute("aria-selected", "true");
    expect(list.getAttribute("aria-activedescendant")).toBe(option(/^Shipper:/).id);

    fireEvent.keyDown(list, { key: "End" });
    expect(option(/^Temperature:/)).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(list, { key: "Escape" });
    expect(list).not.toHaveAttribute("aria-activedescendant");
  });

  it("chooses the first field to check from the tally", () => {
    renderBody();

    fireEvent.click(screen.getByRole("button", { name: "4 to check" }));

    expect(option(/^Bill to:/)).toHaveAttribute("aria-selected", "true");
  });

  it("says so when a page cannot be drawn, and the fields still read", async () => {
    mocks.download.mockRejectedValue(new Error("unavailable"));
    renderBody();

    expect(
      await screen.findByText("Page 1 could not be drawn.", undefined, { timeout: 4000 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Open file" })).toHaveLength(2);
    expect(option(/^Load #:/)).toBeInTheDocument();
  });

  it("asks the agent to draft the shipment, and is drafting until it proposes one", () => {
    renderBody();

    fireEvent.click(screen.getByRole("button", { name: "Create shipment" }));

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      `Create a shipment from Northline_RateCon.pdf (document ${DOCUMENT}), using what was read from it.`,
    );
    expect(useDeskStore.getState().documentAsks[DOCUMENT]).toMatchObject({
      threadId: THREAD,
      intent: "create",
    });
    expect(screen.getByText("Drafting the shipment")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create shipment" })).toBeDisabled();
  });

  it("asks the agent to check the fields it names", () => {
    renderBody();

    expect(
      screen.getByText(
        "Check asks the agent to look again at Bill to, Delivery · stop 2, Delivery · stop 3, and Commodity, and to look for Pro #, Pieces, and Temperature.",
      ),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Check 7 fields" }));

    expect(useDeskStore.getState().asks[THREAD]).toContain(
      `On Northline_RateCon.pdf (document ${DOCUMENT}), check these fields`,
    );
    expect(screen.getByText("The agent is checking the fields")).toBeInTheDocument();
  });

  it("holds its asks while the conversation replies to something else", () => {
    renderBody({ replying: true });

    expect(screen.getByRole("button", { name: "Create shipment" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Check 7 fields" })).toBeDisabled();
  });

  it("opens the approval of the drafted shipment from Review", () => {
    const onOpenArtifact = vi.fn();
    renderBody({
      proposals: [
        {
          id: "aprop_1",
          toolName: "create_shipment",
          arguments: { sourceDocumentId: DOCUMENT },
          status: "Pending",
          createdAt: 10,
        },
      ],
      decisions: [{ id: "aart_decision", kind: "decision_request", proposalId: "aprop_1" }],
      onOpenArtifact,
    });

    expect(screen.getByText("Drafted and waiting on your approval")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Review the draft" }));

    expect(onOpenArtifact).toHaveBeenCalledWith("aart_decision");
    expect(screen.queryByRole("button", { name: "Create shipment" })).not.toBeInTheDocument();
  });

  it("links the shipment once it is created", () => {
    renderBody({
      proposals: [
        {
          id: "aprop_1",
          toolName: "create_shipment",
          arguments: { sourceDocumentId: DOCUMENT },
          status: "Executed",
          createdAt: 10,
          executionResult: {
            action: "Created",
            kind: "shipment",
            name: "S-104260",
            ids: {},
            record: { entityType: "shipment", id: "shp_1" },
          },
        },
      ],
    });

    expect(screen.getByText("Created shipment S-104260")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "S-104260" })).toHaveAttribute(
      "href",
      expect.stringContaining("shp_1"),
    );
    expect(screen.getByRole("link", { name: /Open the shipment/ })).toBeInTheDocument();
  });

  it("says why an approved draft was not created and offers to ask again", () => {
    renderBody({
      proposals: [
        {
          id: "aprop_1",
          toolName: "create_shipment",
          arguments: { sourceDocumentId: DOCUMENT },
          status: "ExecutionFailed",
          executionError: "The customer is inactive.",
          createdAt: 10,
        },
      ],
    });

    expect(
      screen.getByText("It was approved but not created: The customer is inactive."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ask again" })).toBeEnabled();
  });

  it("reads a card saved before stop parts and reasons were kept", async () => {
    renderBody({ payload: legacyPayload() });

    expect(screen.getByRole("group", { name: "Delivery · stop 2" })).toBeInTheDocument();
    expect(
      within(screen.getByRole("group", { name: "Delivery · stop 2" })).getByRole("option", {
        name: /^Stop: Anyco Clothes #425/,
      }),
    ).toHaveTextContent("Low confidence · 64%");
    expect(await screen.findByAltText("Page 1 of rate_confirmation.pdf")).toBeInTheDocument();
  });

  it("draws no page when the file is no longer on record", () => {
    renderBody({ payload: currentPayload({ documentId: "" }) });

    expect(
      screen.getAllByText("The file this was read from is no longer on record."),
    ).not.toHaveLength(0);
    expect(screen.queryByRole("button", { name: "Open file" })).not.toBeInTheDocument();
    expect(mocks.download).not.toHaveBeenCalled();
  });
});
