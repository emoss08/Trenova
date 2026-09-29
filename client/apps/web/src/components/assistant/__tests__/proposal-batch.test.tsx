import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { queries } from "@/lib/queries";
import type { AssistantProposal } from "@/types/assistant";
import { DecisionFollowUpProvider } from "../decision-follow-up";
import { ProposalBatchBar } from "../proposal-batch";
import { batchableProposals } from "../proposal-batches";
import { preview } from "./preview-fixtures";

const decideMyProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposals: (...args: unknown[]) => decideMyProposals(...args),
}));

afterEach(() => {
  cleanup();
  decideMyProposals.mockReset();
});

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "post_invoices",
    arguments: { invoiceIds: ["inv_1"] },
    rationale: "",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    ...overrides,
  };
}

function renderBar(proposals: AssistantProposal[], followUp = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(
    queries.agentPreview.proposal("mine", "aprop_1").queryKey,
    preview({ proposalId: "aprop_1", digest: "sha256:shown" }),
  );

  const view = render(
    <QueryClientProvider client={client}>
      <DecisionFollowUpProvider value={followUp}>
        <ProposalBatchBar proposals={proposals} threadId="athr_1" />
      </DecisionFollowUpProvider>
    </QueryClientProvider>,
  );

  return { ...view, client, followUp };
}

describe("batchableProposals", () => {
  it("groups the waiting proposals of one tool, two or more, outside any plan", () => {
    const batches = batchableProposals([
      proposal({ id: "a", toolName: "post_invoice" }),
      proposal({ id: "b", toolName: "send_invoice" }),
      proposal({ id: "c", toolName: "post_invoice" }),
      proposal({ id: "d", toolName: "post_invoice", status: "Executed", executedAt: 5 }),
      proposal({ id: "e", toolName: "post_invoice", planId: "apl_1", planStep: 1 }),
      proposal({ id: "f", toolName: "send_invoice", hold: { reason: "Paused" } as never }),
    ]);

    expect(batches).toEqual([
      {
        toolName: "post_invoice",
        proposals: [expect.objectContaining({ id: "a" }), expect.objectContaining({ id: "c" })],
      },
    ]);
  });

  it("caps a batch at what one decision takes", () => {
    const many = Array.from({ length: 55 }, (_, index) => proposal({ id: `p${index}` }));

    expect(batchableProposals(many)[0].proposals).toHaveLength(50);
  });
});

describe("ProposalBatchBar", () => {
  it("offers nothing for a single waiting proposal or a mix of tools", () => {
    const { container } = renderBar([
      proposal(),
      proposal({ id: "aprop_2", toolName: "send_invoices" }),
    ]);

    expect(container).toBeEmptyDOMElement();
  });

  it("approves every waiting proposal with the digests of the previews on screen", async () => {
    decideMyProposals.mockResolvedValue([
      { proposalId: "aprop_1", executed: true, error: null, decision: null },
      {
        proposalId: "aprop_2",
        executed: false,
        error: "This change no longer matches what you were shown.",
        decision: null,
      },
    ]);
    const { followUp } = renderBar([proposal(), proposal({ id: "aprop_2" })]);

    await userEvent.click(screen.getByRole("button", { name: "Approve all 2" }));

    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2"], {
      decision: "Accepted",
      previewDigests: [{ proposalId: "aprop_1", digest: "sha256:shown" }],
    });
    expect(await screen.findByText("1 of 2 approved.")).toBeInTheDocument();
    expect(
      screen.getByText(/This change no longer matches what you were shown\./),
    ).toBeInTheDocument();
    expect(followUp).toHaveBeenCalledWith("aprop_1");
  });
});
