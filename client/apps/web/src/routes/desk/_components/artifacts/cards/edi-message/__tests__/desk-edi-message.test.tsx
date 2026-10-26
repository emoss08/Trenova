import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskEdiMessage, DeskEdiMessageArtifact } from "../desk-edi-message";
import { HOUR, NOW, RAW_214, wireInspection, wireMessage } from "./fixtures";

const server = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get: server.get } };
});
// The shipment card is its own chunk; here it stands in as what it was given, with its way back.
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));

/** The partner's 997 that accepted the 214: an inbound message read from an inbound file. */
const ack = wireMessage({
  id: "edimsg_ack",
  documentTypeId: "edidt_x12_997_inbound",
  direction: "Inbound",
  transactionSet: "997",
  shipmentId: "",
  inboundFileId: "ediinf_1",
  interchangeControlNumber: "000000902",
  groupControlNumber: "902",
  transactionControlNumber: "0902",
  segmentCount: 4,
  deliveryStatus: "",
  deliveryAttempts: 0,
  deliverySentAt: null,
  deliveryLastAttemptAt: null,
  ackStatus: "NotExpected",
  ackMessageId: "",
  ackReceivedAt: null,
  generatedAt: NOW - 2 * HOUR + 300,
  payloadSnapshot: {
    transactionSet: "997",
    functionalAck: {
      originalTransactionSet: "214",
      originalTransactionControlNumber: "0001",
      originalGroupControlNumber: "417",
      groupAcknowledgmentCode: "A",
      acceptedTransactionSetCount: 1,
      receivedTransactionSetCount: 1,
    },
  },
});

type Routes = Record<string, () => unknown>;
let routes: Routes;

function route(overrides: Routes = {}) {
  routes = {
    "/edi/messages/edimsg_1/": () => wireMessage(),
    "/edi/messages/edimsg_1/inspect/": () => wireInspection(),
    "/edi/messages/edimsg_ack/": () => ack,
    "/edi/messages/edimsg_ack/inspect/": () =>
      wireInspection({ message: ack as Record<string, unknown> }),
    ...overrides,
  };
}

function draw(props: Partial<Parameters<typeof DeskEdiMessage>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskEdiMessage
          recordId="edimsg_1"
          order={["edimsg_0", "edimsg_1"]}
          {...handlers}
          {...props}
          {...more}
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return {
    ...view,
    ...handlers,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

const asked = (url: string) => server.get.mock.calls.some(([called]) => called === url);

beforeEach(() => {
  seedPermissions({ [Resource.EDI]: Operation.Read, [Resource.Shipment]: Operation.Read });
  route();
  server.get.mockReset().mockImplementation(async (url: string) => {
    const answer = routes[url];
    if (!answer) throw new Error(`Not found: ${url}`);
    return answer();
  });
});

afterEach(() => {
  clearPermissions();
  vi.restoreAllMocks();
});

describe("DeskEdiMessage", () => {
  it("draws the next message's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "Shipment status", level: 2 });

    server.get.mockReturnValue(new Promise(() => {}));
    step("edimsg_0");

    expect(screen.queryByRole("heading", { name: "Shipment status", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the message is read, then draws where it went and what it says", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the EDI message…");
    expect(container.querySelector(".dk-rec-skel .dk-edi-way")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-edi-nest.dk-edi-st")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-edi-stamp")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Shipment status", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(document.querySelector(".dk-rec-kick")).toHaveTextContent(
      /Outbound.*TYSON.*X12 004010/u,
    );
    expect(document.querySelector(".dk-edi-stamp")).toHaveTextContent("214");
    expect(screen.getByText("To Tyson Foods")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Accepted");

    const way = screen.getByRole("list", { name: "Its way" });
    expect(Array.from(way.children).map((stop) => stop.className)).toEqual([
      "dk-edi-stop dk-done dk-travelled",
      "dk-edi-stop dk-done dk-travelled",
      "dk-edi-stop dk-done dk-travelled dk-here",
    ]);
    expect(within(way).getByText("Acknowledged").closest("li")).toHaveAttribute(
      "aria-current",
      "step",
    );
    expect(within(way).getByText("On the first attempt")).toBeInTheDocument();

    const says = section("What it says");
    expect(within(says).getByText("Arrived at delivery")).toBeInTheDocument();
    expect(within(says).getByText("Tyson Foods DC, Springdale, AR 72762")).toBeInTheDocument();
    expect(within(says).getByText("APT-7781")).toBeInTheDocument();

    const envelope = section("Envelope");
    expect(within(envelope).getByText("as2://tyson/inbound")).toBeInTheDocument();
    expect(
      within(envelope)
        .getAllByRole("group")
        .map((layer) => layer.getAttribute("aria-label")),
    ).toEqual(["Interchange 000000417", "Functional group 417", "Transaction set 0001"]);
    expect(await within(envelope).findByText("SE closes it")).toBeInTheDocument();

    const x12 = section("The X12");
    const tape = await within(x12).findByRole("list", { name: "X12 segments" });
    expect(within(tape).getAllByRole("listitem")).toHaveLength(10);
    expect(within(tape).getByText("X1")).toHaveAttribute("title", "Shipment Status Code");
    expect(within(x12).getByText("Nothing flagged")).toHaveClass("dk-ok");
    expect(server.get).toHaveBeenCalledWith("/edi/messages/edimsg_1/inspect/", {
      signal: expect.any(AbortSignal),
    });

    const names = section("Records it names");
    expect(within(names).getByText("S2610-1366")).toBeInTheDocument();
    expect(within(names).getByText("Shipment · BOL BOL-55120")).toBeInTheDocument();
    expect(within(names).getByText("Its acknowledgment")).toBeInTheDocument();

    expect(screen.getByRole("link", { name: "Open message" })).toHaveAttribute(
      "href",
      "/edi/messages?panelType=edit&panelEntityId=edimsg_1",
    );
  });

  it("reads a message received before Lading, which names no version or flow", async () => {
    route({
      "/edi/messages/edimsg_1/": () => wireMessage({ ladingVersionId: null, flowInstanceId: null }),
    });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Shipment status", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("This EDI message could not be read")).toBeNull();
  });

  it("draws the X12 of a message no Lading version wrote", async () => {
    route({
      "/edi/messages/edimsg_1/inspect/": () =>
        wireInspection({
          provenance: { ladingVersionId: null, flowInstanceId: null },
        }),
    });
    draw();

    const x12 = await screen.findByRole("list", { name: "X12 segments" });
    expect(within(x12).getAllByRole("listitem")).toHaveLength(10);
    expect(screen.queryByText(/could not be inspected/u)).toBeNull();
  });

  it("stops the envelope where delivery gave up, says why, and pins each finding to its segment", async () => {
    route({
      "/edi/messages/edimsg_1/": () =>
        wireMessage({
          deliveryStatus: "DeadLettered",
          deliveryAttempts: 5,
          deliveryLastError: "AS2 MDN signature did not verify",
          ackStatus: "Pending",
          ackMessageId: "",
          ackReceivedAt: null,
          diagnosticCount: 2,
        }),
      "/edi/messages/edimsg_1/inspect/": () =>
        wireInspection({
          inspection: {
            diagnostics: [
              {
                severity: "Error",
                code: "X12_INVALID_CODE",
                source: "validation",
                segmentId: "AT7",
                segmentIndex: 5,
                elementPosition: 1,
                message: "Shipment status code is not in the partner's list",
                suggestedFix: "Map X1 to AF",
              },
              {
                severity: "Warning",
                code: "SEPARATOR_REUSED",
                source: "inspection",
                message: "The repetition separator repeats the component separator",
              },
            ],
          },
        }),
    });
    draw();

    expect(await screen.findByText("Dead-lettered")).toHaveClass("dk-rec-st");
    expect(screen.getByText("2 findings")).toBeInTheDocument();

    const way = screen.getByRole("list", { name: "Its way" });
    expect(within(way).getByText("Gave up after 5 attempts").closest("li")).toHaveClass(
      "dk-failed",
      "dk-here",
    );
    const halt = within(section("Its way")).getByRole("note", { name: "Why delivery gave up" });
    expect(within(halt).getByText("AS2 MDN signature did not verify")).toBeInTheDocument();
    expect(within(halt).getByText(/^Attempt 5 · /u)).toBeInTheDocument();
    expect(halt).toHaveClass("dk-pointed");
    expect(halt.style.getPropertyValue("--dk-edi-at")).toBe("0.5");

    const x12 = section("The X12");
    expect(await within(x12).findByText("1 error · 1 warning")).toHaveClass("dk-bad");
    const flagged = within(x12).getByText("Shipment status code is not in the partner's list");
    const segment = flagged.closest('[role="listitem"]');
    expect(segment).toHaveClass("dk-bad");
    expect(within(segment as HTMLElement).getByText("X1")).toHaveClass("dk-flag");
    const mark = within(segment as HTMLElement)
      .getByText("Shipment status code is not in the partner's list")
      .closest(".dk-edi-mark") as HTMLElement;
    expect(mark).toHaveClass("dk-bad");
    expect(within(mark).getByText("AT7, element 1")).toHaveClass("dk-edi-mark-el");
    expect(within(mark).getByText("X12_INVALID_CODE").tagName).toBe("CODE");
    expect(within(mark).getByText("Fix: Map X1 to AF")).toBeInTheDocument();
    expect(
      within(x12)
        .getByText("The repetition separator repeats the component separator")
        .closest(".dk-edi-loose"),
    ).not.toBeNull();
  });

  it("opens the acknowledgment that answered it in its place, and comes back", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /Its acknowledgment/u }));
    expect(
      await screen.findByRole("heading", { name: "Functional acknowledgment", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("From Tyson Foods")).toBeInTheDocument();
    expect(
      within(section("What it says")).getByText("Answers 214 transaction 0001 in group 417"),
    ).toBeInTheDocument();
    expect(within(section("What it says")).getByText("1 of 1 accepted")).toBeInTheDocument();
    expect(
      screen.getByText("Reprocess the file it came in on the EDI inbound files page"),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to Shipment status" }));
    expect(
      await screen.findByRole("heading", { name: "Shipment status", level: 2 }),
    ).toBeInTheDocument();
  });

  it("opens the load it carries in its place", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S2610-1366/u }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Shipment status" }));
    expect(
      await screen.findByRole("heading", { name: "Shipment status", level: 2 }),
    ).toBeInTheDocument();
  });

  it("leaves the load out for someone who may not read loads", async () => {
    seedPermissions({ [Resource.EDI]: Operation.Read });
    draw();

    const names = section(
      (await screen.findByRole("heading", { name: "Records it names", level: 3 })).textContent ??
        "",
    );
    expect(within(names).queryByText("S2610-1366")).toBeNull();
    expect(within(names).getByText("Its acknowledgment")).toBeInTheDocument();
  });

  it("says the raw X12 was purged, and asks for no inspection of it", async () => {
    route({
      "/edi/messages/edimsg_1/": () => wireMessage({ rawX12: "", rawPurgedAt: NOW - HOUR }),
    });
    draw();

    expect(
      await screen.findByText(
        /^The raw X12 was purged .+ under the retention policy; what it said is kept above\.$/u,
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Raw X12 purged")).toBeInTheDocument();
    expect(within(section("What it says")).getByText("Arrived at delivery")).toBeInTheDocument();
    expect(asked("/edi/messages/edimsg_1/inspect/")).toBe(false);
  });

  it("says only that the raw X12 was purged when no decoded copy was kept either", async () => {
    route({
      "/edi/messages/edimsg_1/": () =>
        wireMessage({ rawX12: "", rawPurgedAt: NOW - HOUR, payloadSnapshot: null }),
    });
    draw();

    expect(
      await screen.findByText(/^The raw X12 was purged .+ under the retention policy\.$/u),
    ).toBeInTheDocument();
    expect(
      within(section("What it says")).getByText("No decoded copy of this message was kept."),
    ).toBeInTheDocument();
    expect(screen.queryByText(/what it said is kept above/u)).toBeNull();
  });

  it("says the X12 could not be inspected, and draws the rest", async () => {
    route({
      "/edi/messages/edimsg_1/inspect/": () => {
        throw new Error("The inspector is unavailable");
      },
    });
    draw();

    expect(await screen.findByText("The inspector is unavailable")).toBeInTheDocument();
    expect(within(section("Envelope")).getAllByRole("group")).toHaveLength(3);
    expect(within(section("Envelope")).queryByText("SE closes it")).toBeNull();
  });

  it("copies the raw X12 whole, and says so", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    draw();

    fireEvent.click(await screen.findByRole("button", { name: "Copy X12" }));
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(writeText).toHaveBeenCalledWith(RAW_214);
  });

  it("folds a long tape to its head and the flagged segments, and unfolds it on request", async () => {
    const extra = Array.from({ length: 12 }, (_, index) => ({
      index: index + 10,
      segmentId: "L11",
      name: "Business Instructions and Reference Number",
      type: "body",
      raw: `L11*REF${index}*PO`,
      rawWithTerminator: `L11*REF${index}*PO~`,
      startOffset: 0,
      endOffset: 0,
      elements: [
        {
          position: 1,
          label: "Reference Identification",
          value: `REF${index}`,
          empty: false,
          required: true,
          known: true,
          startOffset: 0,
          endOffset: 0,
          components: [],
        },
      ],
      malformed: false,
    }));
    const base = wireInspection().inspection as { segments: unknown[] };
    route({
      "/edi/messages/edimsg_1/inspect/": () =>
        wireInspection({ inspection: { segments: [...base.segments, ...extra] } }),
    });
    draw();

    const x12 = section(
      (await screen.findByRole("heading", { name: "The X12", level: 3 })).textContent ?? "",
    );
    const tape = await within(x12).findByRole("list", { name: "X12 segments" });
    expect(within(tape).getAllByRole("listitem")).toHaveLength(12);
    expect(within(tape).getByRole("button", { name: "10 segments folded" })).toBeInTheDocument();

    fireEvent.click(within(x12).getByRole("button", { name: "Show all 22 segments" }));
    expect(within(tape).getAllByRole("listitem")).toHaveLength(22);
    expect(within(x12).getByRole("button", { name: "Fold to the first 12" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("says the message could not be read, with the way back", async () => {
    route({
      "/edi/messages/edimsg_1/": () => {
        throw new Error("EDI message not found");
      },
    });
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This EDI message could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("EDI message not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("draws the message as an artifact of its own, with nothing to step through", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <DeskEdiMessageArtifact recordId="edimsg_1" artifact={{} as never} fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Shipment status", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back to the table" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next message" })).toBeNull();
  });
});
