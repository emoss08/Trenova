import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from "vitest";
import { planPreview, preview } from "@/components/assistant/__tests__/preview-fixtures";
import type {
  PlanPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import { toast } from "@trenova/shared/components/ui/toast";
import { DecisionFlow } from "../decision-flow";
import type { PendingDecisionNode } from "../use-pending-decisions";
import { planNode, proposalNode } from "./pending-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchPlanPreview =
  vi.fn<(request: { scope: string; id: string }, options?: unknown) => Promise<PlanPreview>>();
const decideAgentProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideAgentPlan = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideAgentProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let nodes: PendingDecisionNode[] = [];
let recent: unknown[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchPlanPreview: (request: { scope: string; id: string }, options?: unknown) =>
    fetchPlanPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideAgentPlan: (...args: unknown[]) => decideAgentPlan(...args),
  decideAgentProposal: (...args: unknown[]) => decideAgentProposal(...args),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../use-pending-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../use-pending-decisions")>()),
  usePendingDecisions: () => ({
    data: { nodes, totalCount: nodes.length },
    hasNextPage: false,
    isLoading: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
  // A new array each render, the way a refetch hands one back.
  useRecentDecisions: () => ({ data: [...recent] }),
  usePlanSteps: () => ({ data: [] }),
  decideAgentProposals: (...args: unknown[]) => decideAgentProposals(...args),
}));

let info: MockInstance;
let error: MockInstance;
let warning: MockInstance;

beforeEach(() => {
  info = vi.spyOn(toast, "info");
  error = vi.spyOn(toast, "error");
  warning = vi.spyOn(toast, "warning");
});

afterEach(() => {
  info.mockRestore();
  error.mockRestore();
  warning.mockRestore();
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
  decideAgentProposal.mockReset();
  decideAgentPlan.mockReset();
  decideAgentProposals.mockReset();
  nodes = [];
  recent = [];
});

function renderFlow() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionFlow />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const press = (key: string) => act(() => void fireEvent.keyDown(document.body, { key }));
const approveButton = () => screen.getByRole("button", { name: /^Approve/ });

const why =
  "the record changed since this was proposed: the shipment SHP-1 was at version 3 when this was proposed and is at version 5 now, so nothing was changed";

/* AgentDecision as DecideAgentProposal selects it. */
function decisionOf(proposalId: string) {
  return {
    id: `adec_${proposalId}`,
    proposalId,
    decision: "Accepted",
    reasonCode: "approved_from_desk",
    decidedByUserId: "usr_1",
    previewDigest: `sha256:${proposalId}`,
    previewReviewed: true,
    version: 1,
    createdAt: 1_790_000_200,
  };
}

/* A proposal of the queue, as RecentDecisions reads it once it is decided. */
function decidedRow(proposalId: string, status: string, executionError: string) {
  return {
    id: `adec_${proposalId}`,
    decision: "Accepted",
    reasonCode: "approved_from_desk",
    note: "",
    decidedByUserId: "usr_1",
    decidedByName: "Ada",
    decidedAt: 1_790_000_200,
    proposal: { ...proposalNode({ id: proposalId }), status, executionError },
  };
}

/**
 * An approval of a change whose record moved on since it was proposed is
 * recorded but refused: the server runs nothing and answers with `refused`
 * and its reason rather than an error. The queue says so at once, with the
 * server's reason, instead of when the decisions are read again.
 */
describe("DecisionFlow when the server refuses an approval", () => {
  it("says at once that nothing was changed and takes the change off the queue", async () => {
    nodes = [proposalNode({ id: "aprop_1", reviewRequired: true })];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
      }),
    );
    decideAgentProposal.mockResolvedValue({
      proposalId: "aprop_1",
      executed: false,
      refused: true,
      error: why,
      decision: decisionOf("aprop_1"),
    });
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposal).toHaveBeenCalledTimes(1));
    expect(decideAgentProposal).toHaveBeenCalledWith("aprop_1", {
      decision: "Accepted",
      reasonCode: "approved_from_desk",
      previewDigest: "sha256:aprop_1",
    });
    await waitFor(() => expect(info).toHaveBeenCalledTimes(1));
    expect(info).toHaveBeenCalledWith("This changed since you opened it", { description: why });
    expect(error).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Agents will ask here before they change anything."),
    ).toBeInTheDocument();
  });

  it("does not say it again when the decisions are read back with the change failed", async () => {
    nodes = [proposalNode({ id: "aprop_1", reviewRequired: true })];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
      }),
    );
    decideAgentProposal.mockImplementation(async () => {
      recent = [decidedRow("aprop_1", "ExecutionFailed", why)];
      return {
        proposalId: "aprop_1",
        executed: false,
        refused: true,
        error: why,
        decision: decisionOf("aprop_1"),
      };
    });
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("a");

    await waitFor(() => expect(info).toHaveBeenCalledTimes(1));
    // The queue's rows are windowed and jsdom lays nothing out, so the read-back
    // shows in the cleared queue's count of what was decided today.
    expect(
      await screen.findByText(
        "1 thing was decided today. Agents will ask here before they change anything else.",
      ),
    ).toBeInTheDocument();
    expect(info).toHaveBeenCalledTimes(1);
    expect(error).not.toHaveBeenCalled();
  });

  it("says at once that a refused plan ran nothing", async () => {
    const planWhy =
      "nothing ran, because a record the plan changes moved on after it was proposed; step 1: the record changed since this was proposed: the shipment SHP-1 was at version 3 when this was proposed and is at version 5 now, so nothing was changed";
    nodes = [planNode({ id: "apl_1" })];
    fetchPlanPreview.mockResolvedValue(planPreview({ digest: "sha256:plan" }));
    decideAgentPlan.mockResolvedValue({
      refused: true,
      plan: {
        id: "apl_1",
        status: "Failed",
        stepCount: 2,
        completedSteps: 0,
        failedStep: 1,
        failureError: planWhy,
        decidedByUserId: "usr_1",
        decidedAt: 1_790_000_200,
        version: 2,
        updatedAt: 1_790_000_200,
      },
    });
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentPlan).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(info).toHaveBeenCalledWith("This changed since you opened it", {
        description: planWhy,
      }),
    );
    expect(error).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Agents will ask here before they change anything."),
    ).toBeInTheDocument();
  });

  it("takes a refused change of a batch off the queue with the ones that went through", async () => {
    nodes = [
      proposalNode({ id: "aprop_1", reviewRequired: false }),
      proposalNode({ id: "aprop_2", reviewRequired: false, createdAt: 1_790_000_100 }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
      }),
    );
    decideAgentProposals.mockResolvedValue([
      {
        proposalId: "aprop_1",
        executed: true,
        refused: false,
        error: null,
        decision: decisionOf("aprop_1"),
      },
      {
        proposalId: "aprop_2",
        executed: false,
        refused: true,
        error: why,
        decision: decisionOf("aprop_2"),
      },
    ]);
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");
    await waitFor(() => expect(screen.getByRole("button", { name: /^Approve 2/ })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(warning).toHaveBeenCalledWith("1 of 2 went through", { description: why }),
    );
    expect(
      await screen.findByText("Agents will ask here before they change anything."),
    ).toBeInTheDocument();
  });
});
