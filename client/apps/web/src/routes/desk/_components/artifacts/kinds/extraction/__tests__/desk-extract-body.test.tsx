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

/** A value as the load lists it, found by what a screen reader hears for it. */
function value(name: RegExp) {
  return screen.getByRole("button", { name });
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

async function drawnPage(page: number) {
  const image = await screen.findByAltText(`Page ${page} of Northline_RateCon.pdf`);
  fireEvent.load(image);
  return image;
}

const created = {
  id: "aprop_1",
  toolName: "shipment_create",
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
} as Partial<AssistantProposal>;

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
  it("names the load by its lane and leads with the rate exactly as read", () => {
    const { container } = renderBody();

    expect(
      screen.getByRole("heading", { name: "Memphis, TN → Fort Worth, TX", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-who")).toHaveTextContent(
      "Load NL-482913 · Northline Logistics · 3 stops",
    );
    expect(container.querySelector(".dk-rec-stub")).toHaveTextContent("RC");
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("$2,465.00rate as read");
    expect(container.querySelector(".dk-rec-kick")).toHaveTextContent(
      "Rate confirmationNorthline_RateCon.pdf2 pages",
    );
  });

  it("counts what needs a person, what was not read and what was read clearly", () => {
    const { container } = renderBody();

    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("4 to check");
    expect(container.querySelector(".dk-rec-q")).toHaveTextContent("3 not read·7 read clearly");
    expect(container.querySelector(".dk-rec-q")).not.toHaveTextContent("sure of the kind");
  });

  it("says how sure it was of the kind only when it was not sure", () => {
    const { container } = renderBody({ payload: currentPayload({ kindConfidence: 0.62 }) });

    expect(container.querySelector(".dk-rec-q")).toHaveTextContent("62% sure of the kind");
  });

  it("lists what to check with why, beside the bit of page it was read from", async () => {
    renderBody();
    const check = section("Check before it becomes a shipment");

    expect(within(check).getByText("4 to check · 3 not read")).toBeInTheDocument();
    const items = within(check).getAllByRole("listitem");
    expect(items.map((item) => item.querySelector(".dk-ex-ck-l")?.textContent)).toEqual([
      "Bill to",
      "Delivery · stop 2",
      "Delivery · stop 3",
      "Commodity",
      "Pro #",
      "Pieces",
      "Temperature",
    ]);
    expect(items[0]).toHaveTextContent("Northline LogisticsMarked for review · 71%");
    expect(items[1]).toHaveTextContent(
      "Delivery window10/21/2026used10/22/2026Two readings disagree",
    );
    expect(items[2]).toHaveTextContent("NameAnyco Clothes #612Marked for review · 74%");
    expect(items[3]).toHaveTextContent("Paper productsLow confidence · 78%");
    expect(items[4]).toHaveTextContent("Pro #Not readNot found on any page that was read");

    expect(
      await within(items[0]).findByRole("button", { name: "Show Bill to on page 1" }),
    ).toBeInTheDocument();
    expect(
      within(items[2]).getByRole("button", { name: "Show Delivery · stop 3 on page 2" }),
    ).toBeInTheDocument();
    expect(within(items[3]).queryByRole("button", { name: /^Show/ })).toBeNull();
  });

  it("draws the snippet from the page image, with the value outlined where it was read", async () => {
    renderBody();
    const snippet = await screen.findByRole("button", { name: "Show Bill to on page 1" });

    await waitFor(() => expect(snippet.querySelector("img")).not.toBeNull());
    const image = snippet.querySelector("img") as HTMLImageElement;
    const box = snippet.querySelector<HTMLElement>(".dk-ex-snip-box.dk-warn");
    expect(image.style.width).not.toBe("");
    expect(box).not.toBeNull();
    expect(parseFloat(box?.style.left ?? "")).toBeGreaterThan(0);
    expect(mocks.download).toHaveBeenCalledWith(
      documentPageImageEndpoint(DOCUMENT, 1),
      "page.webp",
      { signal: expect.any(AbortSignal) },
    );
  });

  it("lays the load out as the shipment would be: the route, then the facts", () => {
    renderBody();
    const load = section("The load as read");

    const stops = within(within(load).getByRole("list", { name: "Stops" })).getAllByRole(
      "listitem",
    );
    expect(stops.map((stop) => stop.textContent)).toEqual([
      "1PickupAppointmentRidgefield Paper Co.10/20/2026 08:00-12:00",
      "2DeliveryAnyco Clothes DC #425Delivery window10/21/2026",
      "3DeliveryAnyco Clothes #612",
    ]);
    expect(stops[1]).toHaveClass("dk-warn");
    expect(
      [...load.querySelectorAll(".dk-ex-panel h4")].map((heading) => heading.textContent),
    ).toEqual(["Parties", "References", "Equipment and freight", "Charges", "Other"]);
    expect(value(/^Rate: 2465.00/)).toHaveTextContent("$2,465.00");
    expect(value(/^Rate: 2465.00/)).toHaveClass("dk-fair");
    expect(
      value(/^Bill to: Northline Logistics, needs a look, 71 percent confident, page 1$/),
    ).toHaveClass("dk-doubtful");
    expect(within(load).getAllByText("Not read")).toHaveLength(3);
  });

  it("turns to the page a chosen value was read from and marks its thumbnail", async () => {
    renderBody();
    await drawnPage(1);

    fireEvent.click(value(/^Name: Anyco Clothes #612/));

    await drawnPage(2);
    expect(screen.getByRole("button", { name: "Page 2, 1 field to check" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("button", { name: "Page 1, 2 fields to check" })).not.toHaveAttribute(
      "aria-current",
    );
    expect(value(/^Name: Anyco Clothes #612/)).toHaveAttribute("aria-pressed", "true");
  });

  it("boxes each value once the page is drawn, at the fractions of the page it was read at", async () => {
    const { container } = renderBody();

    await waitFor(() =>
      expect(mocks.download).toHaveBeenCalledWith(
        documentPageImageEndpoint(DOCUMENT, 1),
        "page.webp",
        { signal: expect.any(AbortSignal) },
      ),
    );
    expect(container.querySelectorAll(".dk-ex-sheet .dk-ex-box")).toHaveLength(0);

    await drawnPage(1);

    const load = container.querySelector<HTMLElement>('.dk-ex-box[title="Load #: NL-482913"]');
    expect(load).not.toBeNull();
    expect(load?.style.left).toBe("70.75%");
    expect(load?.style.top).toBe("8.5%");
    expect(load?.style.width).toBe("11.25%");
    expect(load?.style.height).toBe("1.4%");
    expect(container.querySelector('.dk-ex-box[title="Bill to: Northline Logistics"]')).toHaveClass(
      "dk-warn",
    );
  });

  it("chooses a value when its box is pressed, and lights the box when its value is pointed at", async () => {
    const { container } = renderBody();
    await drawnPage(1);

    const box = container.querySelector<HTMLElement>(
      '.dk-ex-box[title="Bill to: Northline Logistics"]',
    );
    fireEvent.click(box as HTMLElement);

    expect(value(/^Bill to:/)).toHaveAttribute("aria-pressed", "true");
    expect(box).toHaveClass("dk-on");
    expect(container.querySelector(".dk-ex-flag")).toHaveTextContent("Bill to");
    expect(container.querySelector(".dk-ex-ck.dk-on")).toHaveTextContent("Bill to");

    fireEvent.mouseEnter(value(/^Load #:/));
    expect(container.querySelector('.dk-ex-box[title="Load #: NL-482913"]')).toHaveClass("dk-hot");

    fireEvent.click(value(/^Bill to:/));
    expect(value(/^Bill to:/)).toHaveAttribute("aria-pressed", "false");
  });

  it("brings the document into view when a value is found from its snippet", async () => {
    const scroll = vi.fn();
    Element.prototype.scrollIntoView = scroll;
    renderBody();

    fireEvent.click(
      await screen.findByRole("button", { name: "Show Delivery · stop 3 on page 2" }),
    );

    expect(value(/^Name: Anyco Clothes #612/)).toHaveAttribute("aria-pressed", "true");
    expect(scroll).toHaveBeenCalled();
    expect(scroll.mock.contexts.at(-1)).toHaveClass("dk-ex-doc");
  });

  it("says so when a page cannot be drawn, and the reading still stands", async () => {
    mocks.download.mockRejectedValue(new Error("unavailable"));
    renderBody();

    expect(
      await screen.findByText("Page 1 could not be drawn.", undefined, { timeout: 4000 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Open file" })).toHaveLength(2);
    expect(value(/^Load #:/)).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("button", { name: /^Show / })).toBeNull());
    expect(screen.getAllByRole("listitem").length).toBeGreaterThan(0);
  });

  it("asks the agent to draft the shipment, and is drafting until it proposes one", () => {
    const { container } = renderBody();

    fireEvent.click(screen.getByRole("button", { name: "Create shipment" }));

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      `Create a shipment from Northline_RateCon.pdf (document ${DOCUMENT}), using what was read from it.`,
    );
    expect(useDeskStore.getState().documentAsks[DOCUMENT]).toMatchObject({
      threadId: THREAD,
      intent: "create",
    });
    expect(container.querySelector(".dk-rec-bar")).toHaveTextContent("Drafting the shipment");
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Drafting the shipment");
    expect(screen.getByRole("button", { name: "Create shipment" })).toBeDisabled();
  });

  it("asks the agent to check the fields it names", () => {
    const { container } = renderBody();
    const check = screen.getByRole("button", { name: "Ask the agent to check 7" });

    expect(check).toHaveAttribute(
      "title",
      "Check asks the agent to look again at Bill to, Delivery · stop 2, Delivery · stop 3, and Commodity, and to look for Pro #, Pieces, and Temperature.",
    );
    fireEvent.click(check);

    expect(useDeskStore.getState().asks[THREAD]).toContain(
      `On Northline_RateCon.pdf (document ${DOCUMENT}), check these fields`,
    );
    expect(container.querySelector(".dk-rec-bar")).toHaveTextContent(
      "The agent is checking the fields",
    );
  });

  it("holds its asks while the conversation replies to something else", () => {
    renderBody({ replying: true });

    expect(screen.getByRole("button", { name: "Create shipment" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Ask the agent to check 7" })).toBeDisabled();
  });

  it("opens the approval of the drafted shipment from the bar", () => {
    const onOpenArtifact = vi.fn();
    const { container } = renderBody({
      proposals: [
        {
          id: "aprop_1",
          toolName: "shipment_create",
          arguments: { sourceDocumentId: DOCUMENT },
          status: "Pending",
          createdAt: 10,
        },
      ],
      decisions: [{ id: "aart_decision", kind: "decision_request", proposalId: "aprop_1" }],
      onOpenArtifact,
    });

    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Drafted, waiting on you");
    expect(container.querySelector(".dk-rec-bar")).toHaveTextContent(
      "The agent drafted it; nothing is created until you approve",
    );
    fireEvent.click(screen.getByRole("button", { name: "Review the draft" }));

    expect(onOpenArtifact).toHaveBeenCalledWith("aart_decision");
    expect(screen.queryByRole("button", { name: "Create shipment" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Ask the agent to check/ })).toBeNull();
  });

  it("links the shipment once it is created, and stops asking for checks", () => {
    const { container } = renderBody({ proposals: [created] });

    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Shipment created");
    expect(
      within(container.querySelector(".dk-rec-q") as HTMLElement).getByRole("link", {
        name: "S-104260",
      }),
    ).toHaveAttribute("href", expect.stringContaining("shp_1"));
    expect(container.querySelector(".dk-rec-q")).not.toHaveTextContent("not read");
    expect(screen.getByRole("link", { name: "Open S-104260" })).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-bar")).toHaveTextContent("Created from this reading");
    expect(
      screen.queryByRole("heading", { name: "Check before it becomes a shipment" }),
    ).toBeNull();
  });

  it("says why an approved draft was not created and offers to ask again", () => {
    const { container } = renderBody({
      proposals: [
        {
          id: "aprop_1",
          toolName: "shipment_create",
          arguments: { sourceDocumentId: DOCUMENT },
          status: "ExecutionFailed",
          executionError: "The customer is inactive.",
          createdAt: 10,
        },
      ],
    });

    expect(container.querySelector(".dk-rec-bar")).toHaveTextContent(
      "It was approved but not created: The customer is inactive.",
    );
    expect(screen.getByRole("button", { name: "Ask again" })).toBeEnabled();
  });

  it("reads a card saved before stops named their place, titling it by its kind", async () => {
    const { container } = renderBody({ payload: legacyPayload() });

    expect(
      screen.getByRole("heading", { name: "Rate confirmation", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent("$4,500.00rate as read");
    expect(container.querySelector(".dk-rec-amt b")).toHaveClass("dk-warn");
    const stop = within(section("Check before it becomes a shipment")).getAllByRole("listitem")[0];
    expect(stop).toHaveTextContent("Delivery · stop 2");
    expect(stop).toHaveTextContent("Low confidence · 64%");
    expect(await screen.findByAltText("Page 2 of rate_confirmation.pdf")).toBeInTheDocument();
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
