import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { queries } from "@/lib/queries";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { decisionRequestsFrom, decisionRequestsFromSteps } from "../decision-requests";
import { RequestedDecision } from "../requested-decision";
import type { ToolStep } from "../activity";
import type { ToolExchange } from "../thread-view";

vi.mock("../proposal-card", () => ({
  ProposalCard: ({ proposal }: { proposal: AssistantProposal }) => (
    <div data-testid="proposal-card">{proposal.id}</div>
  ),
}));

vi.mock("../plan-card", () => ({
  PlanCard: ({ plan, steps }: { plan: AssistantPlan; steps: AssistantProposal[] }) => (
    <div data-testid="plan-card">
      {plan.id}:{steps.map((step) => step.id).join(",")}
    </div>
  ),
}));

afterEach(cleanup);

const THREAD = "athr_1";

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    threadId: THREAD,
    sourceMessageId: "amsg_1",
    toolName: "create_shipment",
    arguments: {},
    rationale: "Copy PRO-100 for Tuesday.",
    confidence: 0.8,
    autonomyTier: "Propose",
    status: "Pending",
    planId: "",
    planStep: 0,
    executedAt: null,
    executionError: "",
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  } as AssistantProposal;
}

function renderRequested(
  proposalId: string,
  proposals: AssistantProposal[],
  plans: AssistantPlan[] = [],
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(queries.assistant.proposals(THREAD).queryKey, { results: proposals });
  client.setQueryData(queries.assistant.plans(THREAD).queryKey, { results: plans });

  return render(
    <QueryClientProvider client={client}>
      <RequestedDecision proposalId={proposalId} threadId={THREAD} />
    </QueryClientProvider>,
  );
}

describe("RequestedDecision", () => {
  it("puts the proposal's own card back in front of the person", () => {
    renderRequested("aprop_1", [proposal(), proposal({ id: "aprop_2" })]);

    expect(screen.getByTestId("proposal-card")).toHaveTextContent("aprop_1");
  });

  it("shows a plan's card, with its steps in order, for a step of one", () => {
    renderRequested(
      "aprop_2",
      [
        proposal({ id: "aprop_2", planId: "aplan_1", planStep: 2 }),
        proposal({ id: "aprop_1", planId: "aplan_1", planStep: 1 }),
        proposal({ id: "aprop_3" }),
      ],
      [
        {
          id: "aplan_1",
          runId: "arun_1",
          title: "Void and recreate",
          summary: "",
          status: "Pending",
          stepCount: 2,
          completedSteps: 0,
          failureError: "",
          expiresAt: 0,
          createdAt: 1,
        } as AssistantPlan,
      ],
    );

    expect(screen.getByTestId("plan-card")).toHaveTextContent("aplan_1:aprop_1,aprop_2");
    expect(screen.queryByTestId("proposal-card")).not.toBeInTheDocument();
  });

  it("says so when the proposal is no longer in the conversation", () => {
    renderRequested("aprop_9", [proposal()]);

    expect(
      screen.getByText("This proposal is no longer part of the conversation."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("proposal-card")).not.toBeInTheDocument();
  });
});

function decisionExchange(id: string, proposalId: unknown, failed = false): ToolExchange {
  return {
    call: { id, name: "request_decision", arguments: { proposalId } },
    result: {
      id: `amsg_${id}`,
      threadId: THREAD,
      kind: "Message",
      sequence: 3,
      role: "Tool",
      content: "The card is in front of the person again.",
      toolCallId: id,
      toolName: "request_decision",
      toolFailed: failed,
      refused: false,
      scopeStage: "",
      scopeCategory: "",
      scopeReason: "",
      model: "",
      providerId: "",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: 0,
    } as ToolExchange["result"],
  };
}

describe("decisionRequestsFrom", () => {
  it("reads each card a saved turn put back, once", () => {
    expect(
      decisionRequestsFrom([
        decisionExchange("call_a", "aprop_1"),
        decisionExchange("call_b", "aprop_1"),
        decisionExchange("call_c", "aprop_2", true),
        decisionExchange("call_d", 42),
        { call: { id: "call_e", name: "ask_user", arguments: {} }, result: null },
      ]),
    ).toEqual([{ callId: "call_a", proposalId: "aprop_1" }]);
  });

  it("reads a live turn's finished calls only", () => {
    const step = (id: string, status: ToolStep["status"]): ToolStep => ({
      id,
      name: "request_decision",
      arguments: { proposalId: `aprop_${id}` },
      status,
      content: "",
      effect: "ask",
      summary: "",
      durationSeconds: null,
    });

    expect(
      decisionRequestsFromSteps([step("1", "done"), step("2", "running"), step("3", "failed")]),
    ).toEqual([{ callId: "1", proposalId: "aprop_1" }]);
  });
});
