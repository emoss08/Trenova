import type { ActivityEntry } from "@/lib/graphql/record-activity";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskCarrierSettlement } from "../desk-carrier-settlement";
import {
  DAY,
  emptySettlement,
  HOUR,
  paidSettlement,
  pendingSettlement,
  voidedSettlement,
} from "./fixtures";

const fetchers = vi.hoisted(() => ({
  settlement: vi.fn(),
  activity: vi.fn(),
  shipments: vi.fn(),
  submit: vi.fn(),
  approve: vi.fn(),
  post: vi.fn(),
  recalculate: vi.fn(),
}));

vi.mock("@/lib/graphql/settlement-card", () => ({
  fetchDeskDriverSettlement: vi.fn(),
  fetchDeskCarrierSettlement: fetchers.settlement,
}));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
vi.mock("@/lib/graphql/carrier-settlement", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-settlement")>()),
  submitCarrierSettlement: fetchers.submit,
  approveCarrierSettlement: fetchers.approve,
  postCarrierSettlement: fetchers.post,
  recalculateCarrierSettlement: fetchers.recalculate,
}));
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({ recordId }: { recordId: string }) => <p>Shipment card {recordId}</p>,
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../carrier/desk-carrier", () => ({
  DeskCarrier: ({ recordId }: { recordId: string }) => <p>Carrier card {recordId}</p>,
  DeskCarrierArtifact: () => null,
}));

const NOW_MS = (DAY.oct14 + 15 * HOUR) * 1000;

const shipment = {
  id: "shp_240301",
  proNumber: "S-240301",
  status: "Completed",
  totalChargeAmount: "2410.00",
  moves: [],
} as unknown as RecordShipment;

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: DAY.oct13 + 15 * HOUR,
  changes: { status: { from: "Draft", to: "PendingApproval" } },
  comment: "Submitted for approval",
  principalType: "session_user",
  user: { id: "usr_mia", name: "Mia Chen" },
} as ActivityEntry;

/** Signs the test's person in to read every section and take these settlement steps. */
function grant(operations: number) {
  seedPermissions({
    [Resource.CarrierSettlement]: Operation.Read | operations,
    [Resource.Shipment]: Operation.Read,
    [Resource.Carrier]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
  });
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onBack = vi.fn();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskCarrierSettlement recordId="cstl_88" onBack={onBack} />
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

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW_MS);
  grant(0);
  fetchers.settlement.mockReset().mockResolvedValue(pendingSettlement());
  fetchers.activity.mockReset().mockResolvedValue({
    rows: [entry],
    total: 1,
    endCursor: null,
    hasNextPage: false,
  });
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 1,
    endCursor: null,
    hasNextPage: false,
  });
  for (const action of [fetchers.submit, fetchers.approve, fetchers.post, fetchers.recalculate]) {
    action.mockReset().mockResolvedValue({ id: "cstl_88" });
  }
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskCarrierSettlement", () => {
  it("holds the card's outline while the settlement is read, then draws the carrier and what it is owed", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the settlement…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-stub")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-cset-strip")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-stub")).toHaveTextContent("AP");
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      /^\$10,600\.00net payable · pays \w+, Nov 10$/u,
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Waiting on approval");
    expect(
      within(section("Where it stands")).getByRole("list", { name: "Settlement lifecycle" }),
    ).toHaveTextContent("DraftSubmittedApprovedPostedPaid");
  });

  it("reconciles the carrier's invoices move by move and lists only those that need someone", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    const invoices = section("Carrier invoices");
    expect(within(invoices).getByText("4 of 6 matched")).toBeInTheDocument();
    const strip = within(invoices).getByRole("img");
    expect(strip).toHaveAccessibleName(
      "4 moves Matched, 1 move Off the buy rate, 1 move Not invoiced yet",
    );
    expect(Array.from(strip.children).map((cell) => cell.className)).toEqual([
      "dk-matched",
      "dk-matched",
      "dk-matched",
      "dk-matched",
      "dk-off",
      "dk-none",
    ]);
    expect(container.querySelector(".dk-cset-keys .dk-off")).toHaveTextContent(
      "Off the buy rate1 move · +$85.00",
    );
    expect(
      within(invoices)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "Invoice INV-77141 · S-240303Billed $1,835.00 · buy rate $1,750.00+$85.00Over",
      "No invoice yet · S-240306Costs $1,720.00 on this statementNot invoiced",
    ]);
  });

  it("reads the costs to the net payable and where the money goes", async () => {
    draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    const costs = section("Costs");
    expect(within(costs).getByText("15 lines")).toBeInTheDocument();
    expect(within(costs).getByText("By hand").closest("tr")).toHaveTextContent(
      "Late delivery creditBy handFlat−$85.00",
    );
    expect(costs.querySelector(".dk-rec-stmt-foot")).toHaveTextContent("Net payable$10,600.00");
    const remit = section("Where the money goes");
    expect(remit).toHaveTextContent(
      "Remit toBlue Ridge Freight RemittancePO Box 4410Roanoke, VA 24015",
    );
    expect(remit).toHaveTextContent("MethodACH (manual) · the carrier's preferred");
    expect(remit).toHaveTextContent("TermsNet 30 · pays Nov 10");
  });

  it("shows what the settlement pays for each load in place of the load's charge", async () => {
    draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    const loads = section("Loads it pays");
    expect(await within(loads).findByText(/pays \$1,845\.00/u)).toBeInTheDocument();
    expect(within(loads).queryByText(/\$2,410\.00/u)).toBeNull();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "carrier_settlement", id: "cstl_88" },
      expect.anything(),
      expect.anything(),
    );
  });

  it("opens the carrier in place with the way back to the settlement", async () => {
    draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    fireEvent.click(within(section("Where it stands")).getByText("Open carrier"));
    expect(await screen.findByText("Carrier card car_ridge")).toBeInTheDocument();
  });

  it("offers no step to someone who may only read the settlement", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
    expect(within(bar(container)).getByRole("link", { name: "Open settlement" })).toHaveAttribute(
      "href",
      expect.stringContaining("cstl_88"),
    );
  });

  it("offers nothing on the payable it stepped away from while the next one is read", async () => {
    grant(Operation.Approve | Operation.Reject | Operation.Cancel);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const card = (recordId: string) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskCarrierSettlement
            recordId={recordId}
            order={["cstl_88", "cstl_91"]}
            onSelect={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>
    );
    const { container, rerender } = render(card("cstl_88"));
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    fetchers.settlement.mockReturnValue(new Promise(() => {}));
    rerender(card("cstl_91"));

    expect(screen.queryByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 })).toBeNull();
    expect(screen.queryByRole("button", { name: "Approve $10,600.00" })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("approves a payable waiting on approval and reads it again", async () => {
    grant(Operation.Approve | Operation.Reject | Operation.Cancel);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    expect(
      within(bar(container))
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Send back", "Void", "Approve $10,600.00"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve $10,600.00" }));
    await vi.waitFor(() =>
      expect(fetchers.approve).toHaveBeenCalledWith({ settlementId: "cstl_88" }),
    );
    await vi.waitFor(() => expect(fetchers.settlement).toHaveBeenCalledTimes(2));
  });

  it("voids a payable only with a reason", async () => {
    grant(Operation.Cancel);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Void" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Void settlement" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Void" })).toBeDisabled();
  });

  it("submits a draft and recalculates it from its cost events", async () => {
    grant(Operation.Submit | Operation.Update);
    fetchers.settlement.mockResolvedValue(
      pendingSettlement({ status: "Draft", submittedAt: null, submittedBy: null }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    expect(
      within(bar(container))
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Recalculate", "Submit for approval"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Recalculate" }));
    await vi.waitFor(() =>
      expect(fetchers.recalculate).toHaveBeenCalledWith({ settlementId: "cstl_88" }),
    );
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Submit for approval" }));
    await vi.waitFor(() =>
      expect(fetchers.submit).toHaveBeenCalledWith({ settlementId: "cstl_88" }),
    );
  });

  it("posts an approved payable to the ledger", async () => {
    grant(Operation.Approve);
    fetchers.settlement.mockResolvedValue(
      pendingSettlement({ status: "Approved", approvedAt: DAY.oct13 + 18 * HOUR }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Post to the ledger" }));
    await vi.waitFor(() => expect(fetchers.post).toHaveBeenCalledWith({ settlementId: "cstl_88" }));
  });

  it("asks how a posted payable was paid, starting from the carrier's preferred method", async () => {
    grant(Operation.Update);
    fetchers.settlement.mockResolvedValue(
      pendingSettlement({ status: "Posted", approvedAt: DAY.oct13, postedAt: DAY.oct14 }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Mark $10,600.00 paid" }));
    const dialog = await screen.findByRole("dialog");
    expect(
      within(dialog).getByRole("heading", { name: "Mark settlement paid" }),
    ).toBeInTheDocument();
    expect(dialog).toHaveTextContent("ACH (manual)");
  });

  it("lists who moved a paid payable through each step and leaves the invoices out", async () => {
    grant(Operation.Approve | Operation.Update | Operation.Cancel);
    fetchers.settlement.mockResolvedValue(paidSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    const stands = section("Where it stands");
    expect(within(stands).getByText("Paid Nov 10 · on its pay date")).toBeInTheDocument();
    expect(
      Array.from(stands.querySelectorAll(".dk-rec-signs dt")).map((term) => term.textContent),
    ).toEqual(["Submitted", "Approved", "Posted", "Paid"]);
    expect(within(section("Carrier invoices")).queryAllByRole("listitem")).toHaveLength(0);
    expect(container.querySelector(".dk-rec-st")?.parentElement).toHaveTextContent(
      "Check · CHK 10442",
    );
    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
  });

  it("says why a voided payable was voided and how far it had got", async () => {
    fetchers.settlement.mockResolvedValue(voidedSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    const why = section("Why it was voided");
    expect(why).toHaveTextContent("Duplicate of CS-2610-0087.");
    expect(why).toHaveTextContent("Approved · never posted");
    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$10,600.00");
    expect(section("Where it stood").querySelector(".dk-rec-track")).toHaveClass("dk-off");
    expect(screen.queryByRole("heading", { name: "Where the money goes" })).toBeNull();
  });

  it("says there is nothing payable for a draft with no costs and no remit-to address", async () => {
    fetchers.settlement.mockResolvedValue(emptySettlement());
    draw();
    await screen.findByRole("heading", { name: "Mesa Haulers", level: 2 });

    expect(section("Costs")).toHaveTextContent("Nothing payable");
    expect(section("Where the money goes")).toHaveTextContent(
      "No remit-to address on file — set it on the carrier record.",
    );
    expect(screen.queryByRole("heading", { name: "Carrier invoices" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Loads it pays" })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
  });

  it("leaves out what the person may not read and never asks for it", async () => {
    seedPermissions({ [Resource.CarrierSettlement]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Blue Ridge Freight LLC", level: 2 });

    expect(screen.queryByRole("heading", { name: "Loads it pays" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    expect(screen.queryByText("Open carrier")).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.activity).not.toHaveBeenCalled();
  });

  it("says the settlement could not be read, with the way back", async () => {
    fetchers.settlement.mockRejectedValue(new Error("Settlement not found"));
    const { onBack } = draw();

    expect(await screen.findByText("This settlement could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });
});
