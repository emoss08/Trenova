import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { planPreview, preview } from "@/components/assistant/__tests__/preview-fixtures";
import type { PlanPreview, ProposalPreview } from "@/lib/graphql/agent-preview";
import { DecisionCard, type CardGate } from "../decision-card";
import type { PendingPlanNode, PendingProposalNode, PlanStepNode } from "../use-pending-decisions";
import { planNode, proposalNode } from "./pending-fixtures";

const fetchProposalPreview = vi.fn<(...args: unknown[]) => Promise<ProposalPreview>>();
const fetchPlanPreview = vi.fn<(...args: unknown[]) => Promise<PlanPreview>>();
let steps: PlanStepNode[] | undefined = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (...args: unknown[]) => fetchProposalPreview(...args),
  fetchPlanPreview: (...args: unknown[]) => fetchPlanPreview(...args),
}));

vi.mock("../use-pending-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../use-pending-decisions")>()),
  usePlanSteps: () => ({ data: steps }),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
  steps = [];
});

function renderCard(node: PendingProposalNode | PendingPlanNode) {
  const onGate = vi.fn<(gate: CardGate) => void>();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionCard
          node={node}
          leaving={null}
          onGate={onGate}
          onPreviewShown={vi.fn()}
          onDeclineWith={vi.fn()}
          onModify={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { lastGate: () => onGate.mock.calls.at(-1)?.[0] };
}

describe("DecisionCard and changes that must be reviewed", () => {
  it("offers no approval of a change that must be reviewed when its preview cannot be read", async () => {
    fetchProposalPreview.mockRejectedValue(new Error("preview unavailable"));
    const { lastGate } = renderCard(proposalNode({ reviewRequired: true }));

    expect(
      await screen.findByText(
        "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
      ),
    ).toBeInTheDocument();
    expect(lastGate()).toEqual(expect.objectContaining({ approvable: false, digest: undefined }));
  });

  it("still approves an internal change unreviewed when its preview cannot be read", async () => {
    fetchProposalPreview.mockRejectedValue(new Error("preview unavailable"));
    const { lastGate } = renderCard(proposalNode({ reviewRequired: false }));

    expect(
      await screen.findByText(
        "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
      ),
    ).toBeInTheDocument();
    expect(lastGate()).toEqual(expect.objectContaining({ approvable: true, digest: undefined }));
  });

  it("approves a change that must be reviewed once its preview is in, with its digest", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({ tool: "post_customer_payment", digest: "sha256:paid", reviewRequired: true }),
    );
    const { lastGate } = renderCard(proposalNode({ reviewRequired: true }));

    await waitFor(() =>
      expect(lastGate()).toEqual(
        expect.objectContaining({ approvable: true, digest: "sha256:paid" }),
      ),
    );
  });

  it("offers no approval of a plan with a step that must be reviewed until its preview is in", async () => {
    steps = [
      proposalNode({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true }),
      proposalNode({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: false }),
    ];
    fetchPlanPreview.mockRejectedValue(new Error("preview unavailable"));
    const { lastGate } = renderCard(planNode());

    expect(
      await screen.findByText(
        "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
      ),
    ).toBeInTheDocument();
    expect(lastGate()).toEqual(expect.objectContaining({ approvable: false }));
  });

  it("approves a plan with a step that must be reviewed with the plan's digest", async () => {
    steps = [proposalNode({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true })];
    fetchPlanPreview.mockResolvedValue(
      planPreview({ digest: "sha256:plan", reviewRequired: true }),
    );
    const { lastGate } = renderCard(planNode({ stepCount: 1 }));

    await waitFor(() =>
      expect(lastGate()).toEqual(
        expect.objectContaining({ approvable: true, digest: "sha256:plan" }),
      ),
    );
  });

  it("treats a plan whose steps are not read yet as one that must be reviewed", async () => {
    steps = undefined;
    fetchPlanPreview.mockRejectedValue(new Error("preview unavailable"));
    const { lastGate } = renderCard(planNode());

    await screen.findByText(
      "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
    );
    expect(lastGate()).toEqual(expect.objectContaining({ approvable: false }));
  });
});
