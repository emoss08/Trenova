import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PlanPreview } from "@/lib/graphql/agent-preview";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { PlanCard } from "../plan-card";
import { planPreview, preview } from "./preview-fixtures";

const decideMyPlan = vi.fn(async () => undefined);
const decideAgentPlan = vi.fn(async () => undefined);
const fetchPlanPreview = vi.fn<(...args: unknown[]) => Promise<PlanPreview>>();

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyPlan: (...args: unknown[]) => decideMyPlan(...(args as [])),
  decideAgentPlan: (...args: unknown[]) => decideAgentPlan(...(args as [])),
}));

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchPlanPreview: (...args: unknown[]) => fetchPlanPreview(...args),
}));

beforeEach(() => {
  fetchPlanPreview.mockResolvedValue(planPreview());
});

afterEach(() => {
  cleanup();
  decideMyPlan.mockClear();
  decideAgentPlan.mockClear();
  fetchPlanPreview.mockReset();
});

function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Dispatch coverage: 2 changes",
    summary: "Both open moves get a driver.",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    expiresAt: 0,
    hold: null,
    createdAt: 1,
    ...overrides,
  };
}

function step(planStep: number, overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: `aprop_${planStep}`,
    runId: "arun_1",
    toolName: "flag_for_manual_review",
    arguments: { category: "MissingBOL", severity: "Medium" },
    rationale: "",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "apl_1",
    planStep,
    fields: [],
    ...overrides,
  };
}

function renderCard(value = plan(), steps = [step(1), step(2)]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PlanCard plan={value} steps={steps} threadId="athr_1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const approveAll = () => screen.getByRole("button", { name: /approve all/i });

describe("PlanCard", () => {
  it("takes one answer for every step and lists them in order", () => {
    renderCard();

    expect(screen.getByText("Dispatch coverage: 2 changes")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /approve all/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reject all/i })).toBeInTheDocument();
    const items = screen.getAllByRole("listitem").map((item) => item.textContent ?? "");
    expect(items).toHaveLength(2);
    expect(items[0]).toMatch(/^1\./u);
    expect(items[1]).toMatch(/^2\./u);
  });

  // A plan in the person's own conversation is theirs to answer, with only
  // the assistant: it goes through the self-scoped mutation, never the
  // approver's one the decisions queue and AI Control use.
  it("decides a plan as the person whose conversation raised it", async () => {
    const user = userEvent.setup();
    renderCard();

    await waitFor(() => expect(approveAll()).toBeEnabled());
    await user.click(approveAll());
    await waitFor(() =>
      expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:plan",
      }),
    );

    await user.click(screen.getByRole("button", { name: /reject all/i }));
    await waitFor(() =>
      expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: "sha256:plan",
      }),
    );
    expect(decideAgentPlan).not.toHaveBeenCalled();
    expect(fetchPlanPreview).toHaveBeenCalledWith(
      { scope: "mine", id: "apl_1" },
      expect.anything(),
    );
  });

  // Approving a plan approves every step's preview at once; until they are
  // in there is nothing to approve.
  it("keeps Approve all off until the plan's preview is in", () => {
    fetchPlanPreview.mockReturnValue(new Promise(() => {}));
    renderCard();

    expect(approveAll()).toBeDisabled();
    expect(screen.getByRole("button", { name: /reject all/i })).toBeEnabled();
  });

  // A later step on a record an earlier step changes is shown as that step
  // leaves it, and the step says which.
  it("shows each step's changes with a note where it starts from an earlier step", async () => {
    renderCard();

    expect(await screen.findByText("Uses the record step 1 changes")).toBeInTheDocument();
    expect(screen.getAllByText("Uses the record step 1 changes")).toHaveLength(1);
    expect(screen.getAllByRole("link", { name: "S-1001" })).toHaveLength(2);
  });

  it("turns Approve all off when a step's record changed since the plan was proposed", async () => {
    fetchPlanPreview.mockResolvedValue(
      planPreview({
        stale: true,
        steps: [
          { proposalId: "aprop_1", step: 1, preview: preview({ stale: true }) },
          { proposalId: "aprop_2", step: 2, preview: preview({ proposalId: "aprop_2" }) },
        ],
      }),
    );
    renderCard();

    expect(await screen.findByText("Changed since it was proposed")).toBeInTheDocument();
    expect(approveAll()).toBeDisabled();
    expect(screen.getByRole("button", { name: /reject all/i })).toBeEnabled();
  });

  it("names the switch holding it instead of offering a decision it will refuse", () => {
    renderCard(plan({ hold: { reason: "AgentShadow", agentName: "Dispatch desk" } }));

    expect(screen.queryByRole("button", { name: /approve all/i })).not.toBeInTheDocument();
    expect(
      screen.getByText("On hold: Dispatch desk is in shadow mode in AI Control."),
    ).toBeInTheDocument();
  });

  // "Approved" and "done" are different facts. A plan that stopped says which
  // step stopped it, so the approver knows what is left to do by hand.
  it("says where a plan stopped and which steps never ran", () => {
    renderCard(
      plan({ status: "Failed", completedSteps: 1, failedStep: 2, failureError: "Rate not found" }),
      [
        step(1, { status: "Executed", executedAt: 5 }),
        step(2, { status: "ExecutionFailed", executionError: "Rate not found" }),
      ],
    );

    expect(screen.queryByRole("button", { name: /approve all/i })).not.toBeInTheDocument();
    expect(
      screen.getByText("Approved, but step 2 of 2 did not run and the rest were skipped."),
    ).toBeInTheDocument();
    expect(screen.getByText("Rate not found")).toBeInTheDocument();
  });

  it("collapses to the outcome once rejected", () => {
    renderCard(plan({ status: "Rejected" }), [
      step(1, { status: "Rejected" }),
      step(2, { status: "Rejected" }),
    ]);

    expect(screen.queryByRole("button", { name: /reject all/i })).not.toBeInTheDocument();
    expect(screen.getByText("Rejected. Nothing was changed.")).toBeInTheDocument();
  });
});
