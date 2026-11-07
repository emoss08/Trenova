import type { DispatchBoardMove, DispatchMoveProposal } from "@/lib/graphql/dispatch-console";
import { stubLayout } from "@/test/layout";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MoveQueue, QueueEmpty, type QueueRead } from "../move-queue";

const NOW = 1_800_000_000;

vi.mock("@trenova/shared/hooks/use-now", () => ({ useNow: () => NOW }));

// Written from DispatchBoardMove in dispatch_console.graphqls: every field the contract
// carries, with the nullable ones absent the way the server sends them.
function boardMove(overrides: Partial<DispatchBoardMove> = {}): DispatchBoardMove {
  return {
    moveId: "sm_1",
    shipmentId: "shp_1",
    proNumber: "PRO-1",
    bol: "",
    moveStatus: "New",
    shipmentStatus: "New",
    sequence: 0,
    moveCount: 1,
    loaded: true,
    distance: 356,
    revenue: 1940,
    customerId: "cus_1",
    customerName: "Halvorsen Foods",
    serviceTypeId: null,
    serviceTypeCode: "",
    requiredTractorTypeId: null,
    requiredTrailerTypeId: null,
    temperatureMin: null,
    temperatureMax: null,
    hasHazmat: false,
    hasActiveHold: false,
    urgency: "Now",
    minutesToPickup: -20,
    isCovered: false,
    originStopId: "stp_1",
    originLocationId: "loc_1",
    originName: "Joliet DC",
    originCity: "Joliet",
    originState: "IL",
    originLatitude: null,
    originLongitude: null,
    originWindowStart: NOW - 20 * 60,
    originWindowEnd: NOW + 108 * 60,
    originActualArrival: null,
    destinationStopId: "stp_2",
    destinationLocationId: "loc_2",
    destinationName: "Columbus Crossdock",
    destinationCity: "Columbus",
    destinationState: "OH",
    destinationLatitude: null,
    destinationLongitude: null,
    destinationWindowStart: NOW + 86_400,
    destinationWindowEnd: null,
    assignmentId: null,
    assignedWorkerId: null,
    assignedWorkerName: "",
    assignedTractorId: null,
    assignedTractorCode: "",
    assignedTrailerId: null,
    assignedTrailerCode: "",
    assignmentAckStatus: "",
    previousMoveTrailerId: null,
    coverageType: "none",
    carrierAssignmentId: null,
    assignedCarrierId: null,
    assignedCarrierName: "",
    carrierTotalCost: null,
    liveTender: null,
    pendingProposal: null,
    failedProposal: null,
    planPrice: null,
    ...overrides,
  } as DispatchBoardMove;
}

function proposal(overrides: Partial<DispatchMoveProposal> = {}): DispatchMoveProposal {
  return {
    id: "ap_1",
    workerId: "wrk_1",
    workerName: "Dana Ortiz",
    tractorId: "trc_1",
    trailerId: null,
    confidence: 0.91,
    rationale: "18 empty miles to the pickup",
    createdAt: NOW - 600,
    expiresAt: null,
    stale: false,
    ...overrides,
  };
}

const ok: QueueRead = { isPending: false, isError: false, error: null, refetch: vi.fn() };

function renderQueue(
  moves: DispatchBoardMove[],
  options: { read?: QueueRead; selectedMoveId?: string | null } = {},
) {
  return render(
    <MoveQueue
      moves={moves}
      read={options.read ?? ok}
      selectedMoveId={options.selectedMoveId ?? null}
      departingMoveIds={new Set()}
      onSelectMove={vi.fn()}
      errorState={<p>board failed</p>}
      emptyState={<p>nothing to cover</p>}
    />,
  );
}

let restoreLayout = () => {};

beforeEach(() => {
  // The queue is virtualized, and jsdom lays nothing out.
  restoreLayout = stubLayout(800);
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

describe("MoveQueue reads", () => {
  it("draws the error, not the empty board, when the board could not be read", () => {
    renderQueue([], { read: { ...ok, isError: true, error: new Error("timeout") } });

    expect(screen.getByText("board failed")).toBeInTheDocument();
    expect(screen.queryByText("nothing to cover")).toBeNull();
  });

  it("says there is nothing to cover only after a read that worked", () => {
    renderQueue([]);

    expect(screen.getByText("nothing to cover")).toBeInTheDocument();
  });

  it("draws neither while the first read is still out", () => {
    renderQueue([], { read: { ...ok, isPending: true } });

    expect(screen.queryByText("nothing to cover")).toBeNull();
    expect(screen.queryByText("board failed")).toBeNull();
  });
});

describe("MoveQueue grouping", () => {
  it("groups moves under the server's urgency, most urgent first", () => {
    renderQueue([
      boardMove({ moveId: "sm_later", proNumber: "PRO-LATER", urgency: "Planned" }),
      boardMove({ moveId: "sm_now", proNumber: "PRO-NOW", urgency: "Now" }),
      boardMove({
        moveId: "sm_late",
        proNumber: "PRO-LATE",
        urgency: "Late",
        originWindowEnd: NOW - 25 * 60,
      }),
    ]);

    const text = screen.getByRole("listbox").textContent ?? "";
    expect(text.indexOf("Late")).toBeLessThan(text.indexOf("PRO-LATE"));
    expect(text.indexOf("PRO-LATE")).toBeLessThan(text.indexOf("Due now"));
    expect(text.indexOf("Due now")).toBeLessThan(text.indexOf("PRO-NOW"));
    expect(text.indexOf("PRO-NOW")).toBeLessThan(text.indexOf("Later"));
    expect(text.indexOf("Later")).toBeLessThan(text.indexOf("PRO-LATER"));
  });

  it("reads an open window as open, never as late", () => {
    renderQueue([boardMove()]);

    const row = screen.getByRole("option");
    expect(row).toHaveTextContent("Open · closes in 1h 48m");
    expect(row).not.toHaveTextContent(/late/i);
  });

  it("marks the open move as selected and only that one", () => {
    renderQueue(
      [
        boardMove({ moveId: "sm_a", proNumber: "A" }),
        boardMove({ moveId: "sm_b", proNumber: "B" }),
      ],
      { selectedMoveId: "sm_b" },
    );

    const [a, b] = screen.getAllByRole("option");
    expect(a).toHaveAttribute("aria-selected", "false");
    expect(b).toHaveAttribute("aria-selected", "true");
  });
});

describe("MoveQueue next step", () => {
  it("names Nova's proposed driver and how sure it is", () => {
    renderQueue([boardMove({ pendingProposal: proposal() })]);

    const row = screen.getByRole("option");
    expect(within(row).getByRole("img", { name: "Nova proposes" })).toBeInTheDocument();
    expect(row).toHaveTextContent("Dana Ortiz");
    expect(row).toHaveTextContent("91%");
    expect(row).toHaveTextContent("18 empty miles to the pickup");
  });

  it("says a stale proposal will be replaced rather than offering its reason", () => {
    renderQueue([boardMove({ pendingProposal: proposal({ stale: true }) })]);

    expect(screen.getByRole("option")).toHaveTextContent(
      "The move changed since this was proposed; the next plan replaces it",
    );
  });

  it("puts a hold before Nova's proposal, since nothing can be dispatched", () => {
    renderQueue([boardMove({ hasActiveHold: true, pendingProposal: proposal() })]);

    const row = screen.getByRole("option");
    expect(row).toHaveTextContent("Can't dispatch until the hold is released");
    expect(row).not.toHaveTextContent("Dana Ortiz");
  });

  it("keeps an approval that wasn't carried out on its move, with the reason in words", () => {
    const reason =
      "Move 1, stop 1: Scheduled window end must be before the next stop's scheduled window start";
    renderQueue([
      boardMove({
        failedProposal: {
          id: "ap_9",
          workerId: "wrk_1",
          workerName: "Dana Ortiz",
          reason,
          failedAt: 1_800_000_500,
        },
      }),
    ]);

    const row = screen.getByRole("option");
    expect(row).toHaveTextContent("Approved Dana Ortiz; not assigned");
    expect(row).toHaveTextContent(reason);
    expect(row).not.toHaveTextContent("No proposal yet");
  });

  it("shows a newer proposal over an older approval that failed", () => {
    renderQueue([
      boardMove({
        pendingProposal: proposal(),
        failedProposal: {
          id: "ap_9",
          workerId: "wrk_2",
          workerName: "Ray Kim",
          reason: "Shipment on hold",
          failedAt: 1_800_000_500,
        },
      }),
    ]);

    const row = screen.getByRole("option");
    expect(row).toHaveTextContent("Dana Ortiz");
    expect(row).not.toHaveTextContent("not assigned");
  });

  it("says no proposal is waiting when none is", () => {
    renderQueue([boardMove()]);

    expect(screen.getByRole("option")).toHaveTextContent("No proposal yet");
  });
});

describe("QueueEmpty", () => {
  it("offers to clear a search that emptied the queue", () => {
    const onClearSearch = vi.fn();
    render(<QueueEmpty narrowed includeCovered={false} onClearSearch={onClearSearch} />);

    expect(screen.getByText("No moves match the search")).toBeInTheDocument();
    screen.getByRole("button", { name: "Clear filters" }).click();
    expect(onClearSearch).toHaveBeenCalledOnce();
  });

  it("says every move is covered when nothing narrowed it", () => {
    render(<QueueEmpty narrowed={false} includeCovered={false} onClearSearch={vi.fn()} />);

    expect(screen.getByText("Every move in this window is covered")).toBeInTheDocument();
  });
});
