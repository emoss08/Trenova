import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AssistantProposal } from "@/types/assistant";
import { DecisionFollowUpProvider } from "../decision-follow-up";
import { ProposalCard } from "../proposal-card";
import { preview } from "./preview-fixtures";

const decideMyProposal = vi.fn(async () => undefined);
const decideAgentProposal = vi.fn(async () => undefined);

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...(args as [])),
  decideAgentProposal: (...args: unknown[]) => decideAgentProposal(...(args as [])),
}));

// Approve waits for the preview and sends its digest.
vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: async () => preview({ digest: "sha256:shown" }),
}));

const approveWhenReady = async () => {
  const button = screen.getByRole("button", { name: /approve/i });
  await waitFor(() => expect(button).toBeEnabled());
  await userEvent.click(button);
};

afterEach(() => {
  cleanup();
  decideMyProposal.mockClear();
  decideAgentProposal.mockClear();
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
          <MemoryRouter>
            <ProposalCard proposal={proposal()} threadId="athr_1" />
          </MemoryRouter>
        </DecisionFollowUpProvider>
      </QueryClientProvider>,
    );

    await approveWhenReady();

    await waitFor(() => expect(followUp).toHaveBeenCalledWith("aprop_1"));
    expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
      decision: "Accepted",
      reasonCode: "",
      previewDigest: "sha256:shown",
    });
  });

  // The person who asked may not hold the approver's permission, so the card
  // answers through the self-scoped mutation and never the queue's.
  it("decides as the conversation's owner, never through the approver's mutation", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ProposalCard proposal={proposal()} threadId="athr_1" />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: /reject/i }));

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Rejected", reasonCode: "" }),
      ),
    );
    expect(decideAgentProposal).not.toHaveBeenCalled();
  });

  it("does nothing outside a conversation that can answer", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ProposalCard proposal={proposal()} threadId="athr_1" />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await approveWhenReady();

    await waitFor(() => expect(decideMyProposal).toHaveBeenCalled());
  });
});
