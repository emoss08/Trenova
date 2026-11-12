import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCaseDecision } from "../use-case-decision";
import { caseDecision } from "./case-fixture";

const api = vi.hoisted(() => ({
  requestGraphQL: vi.fn(),
  decideAgentProposal: vi.fn(),
  decideAgentPlan: vi.fn(),
  undoAgentDecision: vi.fn(),
  commitAgentDecisionNow: vi.fn(),
  fetchPlanPreview: vi.fn(),
  fetchProposalPreview: vi.fn(),
}));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: api.requestGraphQL,
}));
vi.mock("@/lib/graphql/agent-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-decisions")>()),
  decideAgentProposal: api.decideAgentProposal,
  decideAgentPlan: api.decideAgentPlan,
}));
vi.mock("@/lib/graphql/desk-case", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-case")>()),
  undoAgentDecision: api.undoAgentDecision,
  commitAgentDecisionNow: api.commitAgentDecisionNow,
}));
vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchPlanPreview: api.fetchPlanPreview,
  fetchProposalPreview: api.fetchProposalPreview,
}));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function planPreview(overrides: Record<string, unknown> = {}) {
  return {
    planId: "apl_1",
    steps: [],
    digest: "digest-1",
    stale: false,
    withheldCount: 0,
    computedAt: 1_800_000_000,
    reviewRequired: true,
    ...overrides,
  };
}

beforeEach(() => {
  for (const mock of Object.values(api)) {
    mock.mockReset();
  }
  api.requestGraphQL.mockResolvedValue({ agentProposals: { edges: [] } });
  api.fetchPlanPreview.mockResolvedValue(planPreview());
  api.decideAgentPlan.mockResolvedValue({
    refused: false,
    plan: { id: "apl_1", status: "Approving", failureError: "" },
  });
  api.undoAgentDecision.mockResolvedValue(true);
  api.commitAgentDecisionNow.mockResolvedValue(true);
});

describe("deciding the plan on a record's case", () => {
  it("approves with the digest of the preview drawn and asks for the undo window", async () => {
    const { result } = renderHook(() => useCaseDecision(caseDecision()), { wrapper });
    await waitFor(() => expect(result.current.blocked).toBe(false));

    await act(() => result.current.approve());

    expect(api.decideAgentPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Accepted",
      reasonCode: "approved_from_case",
      note: undefined,
      previewDigest: "digest-1",
      undoWindow: true,
    });
    expect(result.current.failure).toBe("");
    expect(result.current.busy).toBeNull();
  });

  it("declines at once, with the reason and no undo window", async () => {
    const { result } = renderHook(() => useCaseDecision(caseDecision()), { wrapper });
    await waitFor(() => expect(result.current.blocked).toBe(false));

    await act(() => result.current.decline("The receiver agreed to hold the door."));

    expect(api.decideAgentPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Rejected",
      reasonCode: "declined_from_case",
      note: "The receiver agreed to hold the door.",
      previewDigest: undefined,
      undoWindow: false,
    });
  });

  it("holds the approval while a record the plan changes has moved on", async () => {
    api.fetchPlanPreview.mockResolvedValue(planPreview({ stale: true }));
    const { result } = renderHook(() => useCaseDecision(caseDecision()), { wrapper });

    await waitFor(() => expect(result.current.stale).toBe(true));
    expect(result.current.blocked).toBe(true);
  });

  it("says a refusal in the server's words when the record changed under the approval", async () => {
    api.decideAgentPlan.mockResolvedValue({
      refused: true,
      plan: { id: "apl_1", status: "Failed", failureError: "The stop was rescheduled by hand." },
    });
    const { result } = renderHook(() => useCaseDecision(caseDecision()), { wrapper });
    await waitFor(() => expect(result.current.blocked).toBe(false));

    await act(() => result.current.approve());

    expect(result.current.failure).toBe(
      "It was refused because the record changed: The stop was rescheduled by hand.",
    );
  });

  it("keeps what the server said when the request itself fails", async () => {
    api.decideAgentPlan.mockRejectedValue(new Error("You may not approve changes to shipments."));
    const { result } = renderHook(() => useCaseDecision(caseDecision()), { wrapper });
    await waitFor(() => expect(result.current.blocked).toBe(false));

    await act(() => result.current.approve());

    expect(result.current.failure).toBe("You may not approve changes to shipments.");
    expect(result.current.busy).toBeNull();
  });

  it("takes back and hurries along the plan's own approval", async () => {
    const { result } = renderHook(
      () => useCaseDecision(caseDecision({ phase: "Undoable", commitsAt: 1_800_000_008 })),
      { wrapper },
    );

    await act(() => result.current.undo());
    await act(() => result.current.doNow());

    expect(api.undoAgentDecision).toHaveBeenCalledWith({ planId: "apl_1" });
    expect(api.commitAgentDecisionNow).toHaveBeenCalledWith({ planId: "apl_1" });
    expect(api.fetchPlanPreview).not.toHaveBeenCalled();
  });

  it("decides a single change through the proposal's own approval", async () => {
    api.fetchProposalPreview.mockResolvedValue({ digest: "digest-p", stale: false, changes: [] });
    api.decideAgentProposal.mockResolvedValue({ refused: false, executed: false, error: null });
    const { result } = renderHook(
      () => useCaseDecision(caseDecision({ kind: "Proposal", id: "aprop_9", stepCount: 1 })),
      { wrapper },
    );
    await waitFor(() => expect(result.current.blocked).toBe(false));

    await act(() => result.current.approve());
    await act(() => result.current.undo());

    expect(api.decideAgentProposal).toHaveBeenCalledWith(
      "aprop_9",
      expect.objectContaining({ previewDigest: "digest-p", undoWindow: true }),
    );
    expect(api.undoAgentDecision).toHaveBeenCalledWith({ proposalId: "aprop_9" });
  });

  it("does nothing when the case has no decision", async () => {
    const { result } = renderHook(() => useCaseDecision(null), { wrapper });

    await act(() => result.current.approve());

    expect(api.decideAgentPlan).not.toHaveBeenCalled();
    expect(api.decideAgentProposal).not.toHaveBeenCalled();
    expect(result.current.steps).toEqual([]);
  });
});
