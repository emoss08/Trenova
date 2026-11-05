import type { DispatchBulkAssignResult } from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDispatchActions } from "../use-dispatch-actions";

const graphql = vi.hoisted(() => ({
  assignDispatchMovesGraphQL: vi.fn(),
  unassignDispatchMovesGraphQL: vi.fn(),
}));
const decisions = vi.hoisted(() => ({
  decideAgentProposal: vi.fn(),
  undoMyDecision: vi.fn(),
}));

vi.mock("@/lib/graphql/dispatch-console", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  ...graphql,
}));
vi.mock("@/lib/graphql/agent-decisions", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  ...decisions,
}));

// DispatchBulkAssignResult as dispatch_console.graphqls defines it.
function bulk(
  results: { moveId: string; success: boolean; error?: string }[],
): DispatchBulkAssignResult {
  return {
    succeeded: results.filter((item) => item.success).length,
    failed: results.filter((item) => !item.success).length,
    results: results.map((item) => ({
      moveId: item.moveId,
      success: item.success,
      error: item.error ?? null,
      assignmentId: item.success ? `asn_${item.moveId}` : null,
      findings: [],
    })),
  } as unknown as DispatchBulkAssignResult;
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const previous = {
  moveId: "sm_1",
  primaryWorkerId: "wrk_dana",
  tractorId: "trc_114",
  trailerId: "trl_220",
  reassign: true,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useDispatchActions undo", () => {
  it("puts the previous driver back when a reassign is undone", async () => {
    graphql.assignDispatchMovesGraphQL.mockResolvedValue(bulk([{ moveId: "sm_1", success: true }]));
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.assign([
        {
          input: {
            moveId: "sm_1",
            primaryWorkerId: "wrk_marcus",
            tractorId: "trc_207",
            reassign: true,
          },
          proNumber: "1048813",
          driverName: "Marcus Bell",
          previous,
        },
      ]);
    });

    expect(result.current.outcome?.text).toBe("1048813 reassigned to Marcus Bell");
    expect(result.current.undoLabel).toBe("Put the previous driver back on 1048813");

    await act(async () => {
      await result.current.undo();
    });

    expect(graphql.assignDispatchMovesGraphQL).toHaveBeenLastCalledWith([previous]);
    expect(graphql.unassignDispatchMovesGraphQL).not.toHaveBeenCalled();
    expect(result.current.outcome?.text).toBe("The previous driver is back on 1048813");
    expect(result.current.canUndo).toBe(false);
  });

  it("leaves a move uncovered again when a fresh assign is undone", async () => {
    graphql.assignDispatchMovesGraphQL.mockResolvedValue(bulk([{ moveId: "sm_1", success: true }]));
    graphql.unassignDispatchMovesGraphQL.mockResolvedValue(
      bulk([{ moveId: "sm_1", success: true }]),
    );
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.assign([
        {
          input: { moveId: "sm_1", primaryWorkerId: "wrk_dana", tractorId: "trc_114" },
          proNumber: "1048813",
          driverName: "Dana Ortiz",
          previous: null,
        },
      ]);
    });
    await act(async () => {
      await result.current.undo();
    });

    expect(graphql.unassignDispatchMovesGraphQL).toHaveBeenCalledWith(["sm_1"]);
    expect(result.current.outcome?.text).toBe("1048813 is uncovered again");
  });

  it("says why an undo the server refused did not happen, instead of claiming it did", async () => {
    graphql.assignDispatchMovesGraphQL.mockResolvedValue(bulk([{ moveId: "sm_1", success: true }]));
    graphql.unassignDispatchMovesGraphQL.mockResolvedValue(
      bulk([{ moveId: "sm_1", success: false, error: "The move is already in transit" }]),
    );
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.assign([
        {
          input: { moveId: "sm_1", primaryWorkerId: "wrk_dana", tractorId: "trc_114" },
          proNumber: "1048813",
          driverName: "Dana Ortiz",
          previous: null,
        },
      ]);
    });
    await act(async () => {
      await result.current.undo();
    });

    expect(result.current.outcome).toMatchObject({
      tone: "warn",
      text: "Couldn't undo: The move is already in transit",
    });
  });

  it("reports a partly failed assign as not done, so its pre-flight stays open", async () => {
    graphql.assignDispatchMovesGraphQL.mockResolvedValue(
      bulk([{ moveId: "sm_1", success: false, error: "Driver is out of hours" }]),
    );
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    let assigned = true;
    await act(async () => {
      assigned = await result.current.assign([
        {
          input: { moveId: "sm_1", primaryWorkerId: "wrk_dana", tractorId: "trc_114" },
          proNumber: "1048813",
          driverName: "Dana Ortiz",
        },
      ]);
    });

    expect(assigned).toBe(false);
    expect(result.current.canUndo).toBe(false);
  });
});

describe("useDispatchActions proposals", () => {
  it("approves through the decision path with the digest of the dry run shown", async () => {
    decisions.decideAgentProposal.mockResolvedValue({
      proposalId: "ap_1",
      executed: true,
      refused: false,
      error: null,
      decision: null,
    });
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.approveProposal({
        proposalId: "ap_1",
        moveId: "sm_1",
        proNumber: "1048813",
        driverName: "Dana Ortiz",
        previewDigest: "sha256:abc",
      });
    });

    expect(decisions.decideAgentProposal).toHaveBeenCalledWith("ap_1", {
      decision: "Accepted",
      reasonCode: "approved_from_dispatch_console",
      previewDigest: "sha256:abc",
    });
    expect(result.current.outcome?.text).toBe("1048813 assigned to Dana Ortiz");
    expect(result.current.undoLabel).toBe("Unassign 1048813");
  });

  it("takes back an approval still in its undo window rather than unassigning", async () => {
    decisions.decideAgentProposal.mockResolvedValue({
      proposalId: "ap_1",
      executed: false,
      refused: false,
      error: null,
      decision: null,
    });
    decisions.undoMyDecision.mockResolvedValue(true);
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.approveProposal({
        proposalId: "ap_1",
        moveId: "sm_1",
        proNumber: "1048813",
        driverName: "Dana Ortiz",
      });
    });
    expect(result.current.undoLabel).toBe("Take back the approval for 1048813");

    await act(async () => {
      await result.current.undo();
    });

    expect(decisions.undoMyDecision).toHaveBeenCalledWith({ proposalId: "ap_1" });
    expect(graphql.unassignDispatchMovesGraphQL).not.toHaveBeenCalled();
  });

  it("says a refused approval was not assigned and queues nothing to undo", async () => {
    decisions.decideAgentProposal.mockResolvedValue({
      proposalId: "ap_1",
      executed: false,
      refused: true,
      error: "The move changed since it was proposed",
      decision: null,
    });
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    let approved = true;
    await act(async () => {
      approved = await result.current.approveProposal({
        proposalId: "ap_1",
        moveId: "sm_1",
        proNumber: "1048813",
        driverName: "Dana Ortiz",
      });
    });

    expect(approved).toBe(false);
    expect(result.current.outcome?.text).toBe(
      "1048813 was not assigned: The move changed since it was proposed",
    );
    expect(result.current.canUndo).toBe(false);
  });

  it("declines with the dispatcher's reason as the note", async () => {
    decisions.decideAgentProposal.mockResolvedValue({
      proposalId: "ap_1",
      executed: false,
      refused: false,
      error: null,
      decision: null,
    });
    const { result } = renderHook(() => useDispatchActions(), { wrapper });

    await act(async () => {
      await result.current.declineProposal(
        { proposalId: "ap_1", moveId: "sm_1", proNumber: "1048813", driverName: "Dana Ortiz" },
        "Driver asked for a day near home",
      );
    });

    expect(decisions.decideAgentProposal).toHaveBeenCalledWith("ap_1", {
      decision: "Rejected",
      reasonCode: "declined_from_dispatch_console",
      note: "Driver asked for a day near home",
    });
    await waitFor(() =>
      expect(result.current.outcome?.text).toBe("Declined Nova's driver for 1048813"),
    );
  });
});
