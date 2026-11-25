import { TestRouter } from "@trenova/shared/testing/test-router";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskDispatchPlanBody } from "../desk-dispatch-plan-body";
import { boardMove, planArtifact, threeLoads } from "./fixtures";

const fetchers = vi.hoisted(() => ({ moves: vi.fn(), assign: vi.fn() }));

vi.mock("@/lib/graphql/dispatch-console", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/dispatch-console")>()),
  getDispatchMovesByIdsGraphQL: fetchers.moves,
  assignDispatchMovesGraphQL: fetchers.assign,
}));

function grant(assign: boolean) {
  seedPermissions({
    [Resource.ShipmentMove]: Operation.Read | (assign ? Operation.Assign : 0),
  });
}

function draw(payload: Record<string, unknown> = threeLoads) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <DeskDispatchPlanBody artifact={planArtifact(payload)} />
      </TestRouter>
    </QueryClientProvider>,
  );
}

const openMoves = [boardMove("smv_1", "S-1"), boardMove("smv_2", "S-2"), boardMove("smv_3", "S-3")];

beforeEach(() => {
  fetchers.moves.mockReset();
  fetchers.assign.mockReset();
});

afterEach(() => {
  cleanup();
  clearPermissions();
});

describe("DeskDispatchPlanBody", () => {
  it("draws its outline while it reads which loads are covered now", () => {
    grant(true);
    fetchers.moves.mockReturnValue(new Promise(() => {}));
    draw();

    const status = screen.getByRole("status");
    expect(status).toHaveTextContent("Loading the dispatch plan…");
    expect(status.closest("[aria-busy]")).not.toBeNull();
    expect(fetchers.moves).toHaveBeenCalledWith(["smv_1", "smv_2", "smv_3"], expect.anything());
  });

  it("pairs each load with its driver and says what stops the rest", async () => {
    grant(true);
    fetchers.moves.mockResolvedValue(openMoves);
    draw();

    expect(await screen.findByText("Adam Stephens")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Drivers for 3 loads", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("$1,602")).toBeInTheDocument();
    expect(screen.getByText("Planned, nothing assigned")).toBeInTheDocument();
    expect(screen.getByText("2 of 3")).toBeInTheDocument();
    expect(screen.getByText("61 miles empty · 9.5 h to drive")).toBeInTheDocument();
    expect(screen.getByText("440 miles empty · hours unknown")).toBeInTheDocument();
    expect(
      screen.getByText("No ELD has reported this driver's hours; they are unknown, not zero."),
    ).toBeInTheDocument();
    expect(screen.getByText("Cynthia Daniels")).toBeInTheDocument();
    expect(screen.getByText("No driver can take it")).toBeInTheDocument();
    expect(screen.getByText("60/70-hour cycle exhausted")).toBeInTheDocument();
    expect(screen.getAllByText(/cycle exhausted/)).toHaveLength(1);
    expect(
      screen.getByText(
        "1 load you named was not planned: it is covered, closed or picks up later.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Adam Stephens" })).toHaveAttribute(
      "href",
      "/hr/workers/wrk_1",
    );
  });

  it("assigns every open planned load in one call and redraws them from the moves", async () => {
    grant(true);
    fetchers.moves
      .mockResolvedValueOnce(openMoves)
      .mockResolvedValue([
        boardMove("smv_1", "S-1", {
          isCovered: true,
          assignedWorkerId: "wrk_1",
          moveStatus: "Assigned",
        } as never),
        boardMove("smv_2", "S-2", {
          isCovered: true,
          assignedWorkerId: "wrk_2",
          moveStatus: "Assigned",
        } as never),
        boardMove("smv_3", "S-3"),
      ]);
    fetchers.assign.mockResolvedValue({
      succeeded: 2,
      failed: 0,
      results: [
        { moveId: "smv_1", success: true, assignmentId: "asn_1", error: null, findings: [] },
        { moveId: "smv_2", success: true, assignmentId: "asn_2", error: null, findings: [] },
      ],
    });
    draw();

    fireEvent.click(await screen.findByRole("button", { name: "Assign 2 loads" }));

    await waitFor(() =>
      expect(
        screen.getByText("Assigned", { selector: ".dk-rec-st *, .dk-rec-st" }),
      ).toBeInTheDocument(),
    );
    expect(fetchers.assign).toHaveBeenCalledTimes(1);
    expect(fetchers.assign).toHaveBeenCalledWith([
      { moveId: "smv_1", primaryWorkerId: "wrk_1", tractorId: "trc_1", trailerId: null },
      { moveId: "smv_2", primaryWorkerId: "wrk_2", tractorId: "trc_2", trailerId: "trl_2" },
    ]);
    expect(screen.getByRole("button", { name: "Assign loads" })).toBeDisabled();
  });

  it("says which loads the service refused, by PRO number and never by id", async () => {
    grant(true);
    fetchers.moves.mockResolvedValue(openMoves);
    fetchers.assign.mockResolvedValue({
      succeeded: 1,
      failed: 1,
      results: [
        { moveId: "smv_1", success: true, assignmentId: "asn_1", error: null, findings: [] },
        {
          moveId: "smv_2",
          success: false,
          assignmentId: null,
          error: "Shipment move sm_01M82WQ9CGGH69WSJT33PMXEAB is covered by an external carrier",
          findings: [],
        },
      ],
    });
    draw();

    fireEvent.click(await screen.findByRole("button", { name: "Assign 2 loads" }));

    const refusal = await screen.findByText(/1 load was not assigned:/);
    expect(refusal.textContent).toContain("S-2:");
    expect(refusal.textContent).not.toContain("sm_01M82WQ9CGGH69WSJT33PMXEAB");
  });

  it("offers no assignment to someone who may not assign drivers, and says why", async () => {
    grant(false);
    fetchers.moves.mockResolvedValue(openMoves);
    draw();

    expect(
      await screen.findByText("Assigning drivers takes the move assign permission."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Assign/ })).toBeNull();
  });

  it("names who covers a load now when someone else took it", async () => {
    grant(true);
    fetchers.moves.mockResolvedValue([
      boardMove("smv_1", "S-1", {
        isCovered: true,
        assignedWorkerId: "wrk_7",
        assignedWorkerName: "Lee Park",
      } as never),
      boardMove("smv_2", "S-2"),
      boardMove("smv_3", "S-3"),
    ]);
    draw();

    expect(await screen.findByText("Now covered by Lee Park")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Assign 1 load" })).toBeEnabled();
  });

  it("still draws the plan as it was made when the moves cannot be read", async () => {
    grant(true);
    fetchers.moves.mockRejectedValue(new Error("boom"));
    draw();

    expect(
      await screen.findByText(
        "Which loads are covered now could not be read; this is the plan as it was made.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Adam Stephens")).toBeInTheDocument();
  });
});
