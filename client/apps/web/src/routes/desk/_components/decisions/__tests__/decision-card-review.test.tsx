import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { planPreview } from "@/components/assistant/__tests__/preview-fixtures";
import type { PlanPreview } from "@/lib/graphql/agent-preview";
import { PlanCard, type CardGate } from "../decision-card";
import type { PendingPlanNode, PlanStepNode } from "../use-pending-decisions";
import { planNode, proposalNode } from "./pending-fixtures";

const fetchPlanPreview = vi.fn<(...args: unknown[]) => Promise<PlanPreview>>();
let steps: PlanStepNode[] | undefined = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchPlanPreview: (...args: unknown[]) => fetchPlanPreview(...args),
}));

vi.mock("../use-pending-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../use-pending-decisions")>()),
  usePlanSteps: () => ({ data: steps }),
}));

afterEach(() => {
  fetchPlanPreview.mockReset();
  steps = [];
});

function renderCard(node: PendingPlanNode) {
  const onGate = vi.fn<(gate: CardGate) => void>();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PlanCard node={node} leaving={null} onGate={onGate} onDeclineWith={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { lastGate: () => onGate.mock.calls.at(-1)?.[0] };
}

describe("PlanCard and steps that must be reviewed", () => {
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
