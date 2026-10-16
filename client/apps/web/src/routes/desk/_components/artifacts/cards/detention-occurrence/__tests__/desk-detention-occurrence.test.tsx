import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskDetentionOccurrence } from "../desk-detention-occurrence";
import { ACCRUING_NOW, accruingCharge, at, pendingCharge, waivedCharge } from "./fixtures";

const fetchers = vi.hoisted(() => ({
  detention: vi.fn(),
  shipments: vi.fn(),
  approve: vi.fn(),
  sendNotice: vi.fn(),
  waive: vi.fn(),
  dispute: vi.fn(),
}));

vi.mock("@/lib/graphql/detention-card", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/detention-card")>()),
  fetchDeskDetention: fetchers.detention,
}));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
vi.mock("@/services/api", () => ({
  apiService: {
    detentionService: {
      approve: fetchers.approve,
      sendNotice: fetchers.sendNotice,
      waive: fetchers.waive,
      dispute: fetchers.dispute,
    },
  },
}));
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({ recordId }: { recordId: string }) => <p>Shipment card {recordId}</p>,
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../location/desk-location", () => ({
  DeskLocation: ({ recordId }: { recordId: string }) => <p>Location card {recordId}</p>,
  DeskLocationArtifact: () => null,
}));

const shipment = {
  id: "shp_240193",
  proNumber: "S-240193",
  status: "Completed",
  totalChargeAmount: "3666.12",
  moves: [],
} as unknown as RecordShipment;

/** Signs the test's person in to read every section, and to act on charges when asked. */
function grant(act = false) {
  seedPermissions({
    [Resource.Detention]: Operation.Read | (act ? Operation.Update : 0),
    [Resource.Shipment]: Operation.Read,
    [Resource.Location]: Operation.Read,
    [Resource.Customer]: Operation.Read,
  });
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onBack = vi.fn();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskDetentionOccurrence recordId="dto_kroger" onBack={onBack} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, onBack };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

function bar(container: HTMLElement): HTMLElement {
  const found = container.querySelector<HTMLElement>(".dk-rec-bar");
  if (!found) throw new Error("No bar");
  return found;
}

function buttons(container: HTMLElement): (string | null)[] {
  return within(bar(container))
    .queryAllByRole("button")
    .map((button) => button.textContent);
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(at(16) * 1000);
  grant();
  fetchers.detention.mockReset().mockResolvedValue(pendingCharge());
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 1,
    endCursor: null,
    hasNextPage: false,
  });
  for (const action of [fetchers.approve, fetchers.sendNotice, fetchers.waive, fetchers.dispute]) {
    action.mockReset().mockResolvedValue({ id: "dto_kroger" });
  }
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskDetentionOccurrence", () => {
  it("holds the card's outline while the charge is read, then draws the facility and the charge", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the detention charge…");
    expect(container.querySelector(".dk-rec-skel .dk-dtn-clock")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$337.50to bill · 4h 30m at $75.00/hr",
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Waiting on approval");
    expect(
      within(section("Where it stands")).getByRole("list", { name: "Detention lifecycle" }),
    ).toHaveTextContent("At the dockWaiting on approvalApprovedBilled");
    expect(fetchers.detention).toHaveBeenCalledWith("dto_kroger", expect.anything());
  });

  it("reads the clock, the receipt, the split, the notice and whether it holds up", async () => {
    draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    const clock = section("The clock");
    expect(within(clock).getByText("6h 18m at the dock")).toBeInTheDocument();
    expect(
      within(clock).getByRole("list", { name: "The stop's moments" }).querySelectorAll("li.dk-low"),
    ).toHaveLength(1);
    expect(clock.querySelector(".dk-dtn-sum")).toHaveTextContent(
      "At the dock6h 18mFree time2hBillable4h 18m → 4h 30m",
    );

    const receipt = section("How it was worked out");
    expect(within(receipt).getAllByRole("listitem")).toHaveLength(8);
    expect(receipt.querySelector(".dk-rec-stmt-foot")).toHaveTextContent("Charge$337.50");

    expect(section("Who pays, who's paid")).toHaveTextContent("Worker's detention pay$130.00");
    expect(section("Notice to the customer")).toHaveTextContent(
      "Warning notice · DeliveredTo dock-atl@kroger.com and ap@kroger.com",
    );
    const proof = section("Will it hold up");
    expect(within(proof).getByRole("meter", { name: "Collectability" })).toHaveAttribute(
      "aria-valuenow",
      "86",
    );
    expect(within(proof).getByText("3 records · chain verified")).toBeInTheDocument();
  });

  it("opens the shipment and the facility in its place, and comes back", async () => {
    draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "shipment", id: "shp_240193" },
      expect.anything(),
      expect.anything(),
    );
    fireEvent.click(
      within(section("The stop")).getByRole("button", { name: /Kroger Atlanta DC/u }),
    );
    expect(await screen.findByText("Location card loc_kroger_atl")).toBeInTheDocument();
  });

  it("offers no step to someone who may only read the charge", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    expect(buttons(container)).toEqual([]);
    expect(
      within(bar(container)).getByRole("link", { name: "Open in detention desk" }),
    ).toHaveClass("dk-ink");
  });

  it("approves a charge waiting on approval and reads it again", async () => {
    grant(true);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    expect(buttons(container)).toEqual(["Record dispute", "Waive", "Approve $337.50"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve $337.50" }));
    await vi.waitFor(() => expect(fetchers.approve).toHaveBeenCalledWith("dto_kroger"));
    await vi.waitFor(() => expect(fetchers.detention).toHaveBeenCalledTimes(2));
  });

  it("offers nothing on the charge it stepped away from while the next one is read", async () => {
    grant(true);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const card = (recordId: string) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskDetentionOccurrence
            recordId={recordId}
            order={["dto_kroger", "dto_lineage"]}
            onSelect={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>
    );
    const { container, rerender } = render(card("dto_kroger"));
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    fetchers.detention.mockReturnValue(new Promise(() => {}));
    rerender(card("dto_lineage"));

    expect(screen.queryByRole("heading", { name: "Kroger Atlanta DC", level: 2 })).toBeNull();
    expect(screen.queryByRole("button", { name: "Approve $337.50" })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("asks why before waiving a charge", async () => {
    grant(true);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Waive" }));
    expect(await screen.findByRole("heading", { name: "Waive this charge" })).toBeInTheDocument();
    expect(fetchers.waive).not.toHaveBeenCalled();
  });

  it("warns of an overdue notice on a running clock and sends it", async () => {
    grant(true);
    vi.setSystemTime(ACCRUING_NOW * 1000);
    fetchers.detention.mockResolvedValue(accruingCharge());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Lineage Logistics Dallas", level: 2 });

    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$56.25so far · billing since 8:00 AM",
    );
    expect(within(section("Notice to the customer")).getByRole("alert")).toHaveTextContent(
      "Notice overdue by 1h 8m",
    );
    expect(screen.queryByRole("heading", { name: "How it was worked out" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Who pays, who's paid" })).toBeNull();
    expect(buttons(container)).toEqual(["Waive", "Send notice"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Send notice" }));
    await vi.waitFor(() => expect(fetchers.sendNotice).toHaveBeenCalledWith("dto_lineage"));
  });

  it("says why a waived charge was waived and strikes its amount", async () => {
    grant(true);
    fetchers.detention.mockResolvedValue(waivedCharge());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Home Depot RDC Atlanta", level: 2 });

    expect(section("Why it was waived")).toHaveTextContent("Waived for the Q4 volume commitment.");
    expect(section("Why it was waived")).toHaveTextContent("Waived byDana Ruiz");
    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$187.50");
    expect(section("Where it stood").querySelector(".dk-rec-track")).toHaveClass("dk-off");
    expect(buttons(container)).toEqual(["Record dispute"]);
  });

  it("leaves out what the person may not read and never asks for it", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Kroger Atlanta DC", level: 2 });

    expect(screen.queryByRole("heading", { name: "The shipment" })).toBeNull();
    expect(screen.queryByText("Open shipment")).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    const stop = section("The stop");
    expect(within(stop).queryAllByRole("button")).toHaveLength(0);
    expect(within(stop).queryAllByRole("link")).toHaveLength(0);
    expect(stop).toHaveTextContent("LocationKroger Atlanta DCForest Park, GA");
  });

  it("says the charge could not be read and offers the way back", async () => {
    fetchers.detention.mockRejectedValue(new Error("not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This detention charge could not be read" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });
});
