import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AssistantProposal } from "@/types/assistant";
import { DecisionFollowUpProvider } from "../decision-follow-up";
import { ProposalCard } from "../proposal-card";

const decideProposal = vi.fn(async () => undefined);

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      decideProposal: (...args: unknown[]) => decideProposal(...(args as [])),
    },
  },
}));

afterEach(() => {
  cleanup();
  decideProposal.mockClear();
});

function proposal(): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    threadId: "athr_1",
    sourceMessageId: "amsg_1",
    toolName: "create_dashboard",
    arguments: { name: "Operations", tiles: [] },
    rationale: "A page for the morning numbers.",
    confidence: 0.9,
    autonomyTier: "Propose",
    status: "Pending",
    executedAt: null,
    executionError: "",
    createdAt: 1,
    updatedAt: 1,
  } as unknown as AssistantProposal;
}

/**
 * Approving a proposal asks the agent to say what happened.
 *
 * An approval used to end the conversation: the card flipped to done and the
 * agent said nothing, so the person who approved a dashboard could not tell
 * whether it existed or where to find it.
 */
describe("a decision in the thread", () => {
  it("asks for the turn that follows it, once the decision is recorded", async () => {
    const followUp = vi.fn();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <DecisionFollowUpProvider value={followUp}>
          <ProposalCard proposal={proposal()} threadId="athr_1" />
        </DecisionFollowUpProvider>
      </QueryClientProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: /approve/i }));

    await waitFor(() => expect(followUp).toHaveBeenCalledWith("aprop_1"));
    expect(decideProposal).toHaveBeenCalledWith("aprop_1", "Accepted");
  });

  it("does nothing outside a conversation that can answer", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <ProposalCard proposal={proposal()} threadId="athr_1" />
      </QueryClientProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: /approve/i }));

    await waitFor(() => expect(decideProposal).toHaveBeenCalled());
  });
});
