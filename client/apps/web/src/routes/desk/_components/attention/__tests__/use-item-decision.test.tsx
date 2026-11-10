import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { watchtowerItem } from "../../watchtower/__tests__/item-fixture";
import { useItemDecision } from "../use-item-decision";

const api = vi.hoisted(() => ({
  requestGraphQL: vi.fn(),
  decideAgentProposal: vi.fn(),
  decideAgentPlan: vi.fn(),
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

/** One AgentProposal as PendingProposalFields selects it; unmasked, as the fragment reader returns it. */
function proposalNode(overrides: Record<string, unknown> = {}) {
  return {
    id: "aprop_1",
    reviewRequired: false,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    runId: "arun_1",
    toolName: "invoice_post",
    toolParams: { invoiceId: "inv_1" },
    confidence: 0.9,
    rationale: "Delivered and signed for; the invoice is ready to post.",
    autonomyTier: "RequireApproval",
    status: "Pending",
    planId: null,
    planStep: 0,
    simulatedAt: null,
    simulation: null,
    parameterFields: [],
    modifications: null,
    hold: null,
    evidence: [],
    version: 1,
    createdAt: 1_800_000_000,
    updatedAt: 1_800_000_000,
    run: null,
    ...overrides,
  };
}

function decisionItem(overrides: Partial<DeskAttentionItem> = {}): DeskAttentionItem {
  return {
    ...watchtowerItem({ sourceKind: "AgentProposal", sourceId: "aprop_1" }),
    deskLens: "Decisions",
    aboutId: "inv_1",
    ...overrides,
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  api.requestGraphQL.mockReset();
  api.decideAgentProposal.mockReset();
  api.decideAgentPlan.mockReset();
  api.requestGraphQL.mockResolvedValue({ agentProposals: { edges: [{ node: proposalNode() }] } });
});

describe("deciding a change from Needs you", () => {
  it("reads the one proposal the item stands for", async () => {
    const { result } = renderHook(() => useItemDecision(decisionItem()), { wrapper });

    await waitFor(() => expect(result.current.proposals).toHaveLength(1));
    expect(api.requestGraphQL.mock.calls[0][0].variables.input.fieldFilters).toEqual([
      { field: "id", operator: "eq", value: "aprop_1" },
    ]);
    expect(result.current.reviewed).toBe(false);
  });

  it("approves it as the decisions queue does and says it is done once the server answers", async () => {
    api.decideAgentProposal.mockResolvedValue({
      proposalId: "aprop_1",
      executed: true,
      refused: false,
      error: null,
      decision: null,
    });
    const { result } = renderHook(() => useItemDecision(decisionItem()), { wrapper });
    await waitFor(() => expect(result.current.proposals).toHaveLength(1));

    await act(() => result.current.approve());

    expect(api.decideAgentProposal).toHaveBeenCalledWith("aprop_1", {
      decision: "Accepted",
      reasonCode: "approved_from_needs",
      note: undefined,
      previewDigest: undefined,
    });
    expect(result.current.phase).toBe("done");
  });

  it("says plainly when an approval is refused, in the server's words", async () => {
    api.decideAgentProposal.mockResolvedValue({
      proposalId: "aprop_1",
      executed: false,
      refused: true,
      error: "the invoice is already posted",
      decision: null,
    });
    const { result } = renderHook(() => useItemDecision(decisionItem()), { wrapper });
    await waitFor(() => expect(result.current.proposals).toHaveLength(1));

    await act(() => result.current.approve());

    expect(result.current.phase).toBe("failed");
    expect(result.current.failure).toBe("the invoice is already posted");
  });

  it("declines with the reason the person gave, for the agent to read", async () => {
    api.decideAgentProposal.mockResolvedValue({
      proposalId: "aprop_1",
      executed: false,
      refused: false,
      error: null,
      decision: null,
    });
    const { result } = renderHook(() => useItemDecision(decisionItem()), { wrapper });
    await waitFor(() => expect(result.current.proposals).toHaveLength(1));

    await act(() => result.current.decline("Wait for the corrected rate"));

    expect(api.decideAgentProposal).toHaveBeenCalledWith(
      "aprop_1",
      expect.objectContaining({
        decision: "Rejected",
        reasonCode: "declined_from_needs",
        note: "Wait for the corrected rate",
      }),
    );
    expect(result.current.phase).toBe("declined");
  });

  it("reads every step of a plan and decides the plan as one", async () => {
    api.requestGraphQL.mockResolvedValue({
      agentProposals: {
        edges: [
          { node: proposalNode({ id: "aprop_1", planId: "apln_1", planStep: 1 }) },
          { node: proposalNode({ id: "aprop_2", planId: "apln_1", planStep: 2 }) },
        ],
      },
    });
    api.decideAgentPlan.mockResolvedValue({
      refused: false,
      plan: { id: "apln_1", status: "Completed", failureError: "" },
    });
    const item = decisionItem({ sourceKind: "AgentPlan", sourceId: "apln_1" });
    const { result } = renderHook(() => useItemDecision(item), { wrapper });
    await waitFor(() => expect(result.current.proposals).toHaveLength(2));

    await act(() => result.current.approve());

    expect(api.requestGraphQL.mock.calls[0][0].variables.input.fieldFilters).toEqual([
      { field: "planId", operator: "eq", value: "apln_1" },
    ]);
    expect(api.decideAgentPlan).toHaveBeenCalledWith(
      "apln_1",
      expect.objectContaining({ decision: "Accepted" }),
    );
    expect(result.current.phase).toBe("done");
  });
});
