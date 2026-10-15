import type { ActivityEntry } from "@/lib/graphql/record-activity";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskDriverSettlement } from "../desk-driver-settlement";
import {
  DAY,
  emptySettlement,
  HOUR,
  paidSettlement,
  pendingSettlement,
  shortSettlement,
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
  fetchDeskDriverSettlement: fetchers.settlement,
  fetchDeskCarrierSettlement: vi.fn(),
}));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
vi.mock("@/lib/graphql/driver-settlement", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/driver-settlement")>()),
  submitDriverSettlement: fetchers.submit,
  approveDriverSettlement: fetchers.approve,
  postDriverSettlement: fetchers.post,
  recalculateDriverSettlement: fetchers.recalculate,
}));
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({ recordId }: { recordId: string }) => <p>Shipment card {recordId}</p>,
  DeskShipmentArtifact: () => null,
}));

const NOW_MS = (DAY.oct14 + 15 * HOUR) * 1000;

const shipment = {
  id: "shp_240193",
  proNumber: "S-240193",
  status: "Completed",
  totalChargeAmount: "3666.12",
  moves: [],
} as unknown as RecordShipment;

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: DAY.oct13 + 16 * HOUR,
  changes: { status: { from: "Draft", to: "PendingApproval" } },
  comment: "Submitted for approval",
  principalType: "session_user",
  user: { id: "usr_mia", name: "Mia Chen" },
} as ActivityEntry;

/** Signs the test's person in to read every section and take these settlement steps. */
function grant(operations: number) {
  seedPermissions({
    [Resource.DriverSettlement]: Operation.Read | operations,
    [Resource.Shipment]: Operation.Read,
    [Resource.SettlementDispute]: Operation.Read,
    [Resource.Worker]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
  });
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onBack = vi.fn();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskDriverSettlement recordId="dstl_142" onBack={onBack} />
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
    action.mockReset().mockResolvedValue({ id: "dstl_142" });
  }
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskDriverSettlement", () => {
  it("holds the card's outline while the settlement is read, then draws who it pays and what", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the settlement…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-stub")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-pay-fall")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      /^\$1,684\.20net pay · pays \w+, Oct 17$/u,
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Waiting on approval");
    expect(
      within(section("Where it stands")).getByRole("list", { name: "Settlement lifecycle" }),
    ).toHaveTextContent("DraftSubmittedApprovedPostedPaid");
  });

  it("reads the flags, how the pay adds up, the statement and the worker's dispute", async () => {
    draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(
      within(section("Review before approving"))
        .getAllByRole("listitem")
        .map((item) => item.querySelector("b")?.textContent),
    ).toEqual(["Pay added by hand", "Far from this worker's usual pay"]);
    expect(
      within(section("How it adds up"))
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "Earnings9 loads+$1,886.80",
      "Reimbursements+$127.40",
      "Deductions−$330.00",
      "Net pay$1,684.20",
    ]);
    const pay = section("Pay");
    expect(within(pay).getByText("29 lines")).toBeInTheDocument();
    expect(within(pay).getByText("By hand").closest("tr")).toHaveTextContent(
      "Layover - Detroit 10/8By handFlat$150.00",
    );
    expect(section("Dispute")).toHaveTextContent("Missing pay · Stop pay · S-240201In review");
  });

  it("shows what the settlement pays for each load in place of the load's charge", async () => {
    draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    const loads = section("Loads it pays");
    expect(await within(loads).findByText(/pays \$612\.20/u)).toBeInTheDocument();
    expect(within(loads).queryByText(/\$3,666\.12/u)).toBeNull();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "driver_settlement", id: "dstl_142" },
      expect.anything(),
      expect.anything(),
    );
  });

  it("offers no step to someone who may only read the settlement", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
    expect(within(bar(container)).getByRole("link", { name: "Open settlement" })).toHaveClass(
      "dk-ink",
    );
  });

  it("approves a settlement waiting on approval and reads it again", async () => {
    grant(Operation.Approve | Operation.Reject | Operation.Cancel);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(
      within(bar(container))
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Send back", "Void", "Approve $1,684.20"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve $1,684.20" }));
    await vi.waitFor(() =>
      expect(fetchers.approve).toHaveBeenCalledWith({ settlementId: "dstl_142" }),
    );
    await vi.waitFor(() => expect(fetchers.settlement).toHaveBeenCalledTimes(2));
  });

  it("sends a settlement back only with a reason", async () => {
    grant(Operation.Reject);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(within(bar(container)).queryByRole("button", { name: "Approve $1,684.20" })).toBeNull();
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Send back" }));
    expect(await screen.findByRole("heading", { name: "Reject settlement" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reject" })).toBeDisabled();
  });

  it("submits a draft and recalculates it from its pay events", async () => {
    grant(Operation.Submit | Operation.Update);
    fetchers.settlement.mockResolvedValue(shortSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(
      within(bar(container))
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Recalculate", "Submit for approval"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Recalculate" }));
    await vi.waitFor(() =>
      expect(fetchers.recalculate).toHaveBeenCalledWith({ settlementId: "dstl_147" }),
    );
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Submit for approval" }));
    await vi.waitFor(() =>
      expect(fetchers.submit).toHaveBeenCalledWith({ settlementId: "dstl_147" }),
    );
  });

  it("asks how a posted settlement was paid before marking it paid", async () => {
    grant(Operation.Update);
    fetchers.settlement.mockResolvedValue(
      pendingSettlement({ status: "Posted", postedAt: DAY.oct14, approvedAt: DAY.oct14 }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Mark $1,684.20 paid" }));
    expect(
      await screen.findByRole("heading", { name: "Mark settlement paid" }),
    ).toBeInTheDocument();
  });

  it("hangs a shortfall below zero and flags it critical", async () => {
    fetchers.settlement.mockResolvedValue(shortSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$0.00net pay · $212.40 carries forward",
    );
    expect(within(section("Review before approving")).getByRole("listitem")).toHaveClass(
      "dk-critical",
    );
    const fall = within(section("How it adds up")).getAllByRole("listitem");
    expect(fall.at(-1)).toHaveTextContent("Carries forwardto the next settlement−$212.40");
    expect(fall.at(-1)?.querySelector(".dk-zero")).not.toBeNull();
  });

  it("lists who moved a paid settlement through each step and offers nothing more", async () => {
    grant(Operation.Approve | Operation.Update | Operation.Cancel);
    fetchers.settlement.mockResolvedValue(paidSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    const stands = section("Where it stands");
    expect(within(stands).getByText("Paid Oct 10 · on its pay date")).toBeInTheDocument();
    expect(
      Array.from(stands.querySelectorAll(".dk-rec-signs dt")).map((term) => term.textContent),
    ).toEqual(["Submitted", "Approved", "Posted", "Paid"]);
    expect(screen.queryByRole("heading", { name: "Review before approving" })).toBeNull();
    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
  });

  it("says why a voided settlement was voided and how far it had got", async () => {
    fetchers.settlement.mockResolvedValue(voidedSettlement());
    const { container } = draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    const why = section("Why it was voided");
    expect(why).toHaveTextContent("Generated twice for the same week; DS-2610-0142 stands.");
    expect(why).toHaveTextContent("Approved · never posted");
    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$1,684.20");
    expect(section("Where it stood").querySelector(".dk-rec-track")).toHaveClass("dk-off");
  });

  it("says there is nothing to pay for a draft with no lines", async () => {
    fetchers.settlement.mockResolvedValue(emptySettlement());
    draw();
    await screen.findByRole("heading", { name: "Jordan Lee", level: 2 });

    expect(section("Pay")).toHaveTextContent("Nothing to pay");
    expect(screen.queryByRole("heading", { name: "How it adds up" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Loads it pays" })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
  });

  it("leaves out what the person may not read and never asks for it", async () => {
    seedPermissions({ [Resource.DriverSettlement]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Rosa Martinez", level: 2 });

    expect(screen.queryByRole("heading", { name: "Dispute" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Loads it pays" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    expect(screen.queryByText("Open worker")).toBeNull();
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
