import { preview } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskDecisionBody } from "../desk-decision-body";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const decideMyProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let listed: AssistantProposal[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: vi.fn(),
  decideMyProposals: (...args: unknown[]) => decideMyProposals(...args),
  decideMyPlan: vi.fn(),
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/lib/queries/assistant")>();
  return {
    ...original,
    assistant: {
      ...original.assistant,
      proposals: (threadId: string) => ({
        queryKey: ["assistant-proposals", threadId],
        queryFn: async () => ({ results: listed }),
      }),
    },
  };
});

afterEach(() => {
  cleanup();
  fetchProposalPreview.mockReset();
  decideMyProposals.mockReset();
  listed = [];
});

/* services.AssistantProposal as the thread's proposals route serves it. */
function proposal(overrides: Partial<AssistantProposal>): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "post_customer_payment",
    arguments: {},
    rationale: "",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    createdAt: 100,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    reviewRequired: true,
    ...overrides,
  };
}

function renderBody(ids: string[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const artifact = {
    id: "aart_1",
    threadId: "athr_1",
    kind: "decision_request",
    title: "Post these payments",
    proposalId: ids[0],
    payload: { proposalIds: ids },
  } as unknown as AssistantArtifact;
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DeskDecisionBody artifact={artifact} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const approve = () => screen.getByRole("button", { name: /^Approve/ });

describe("DeskDecisionBody approving changes together", () => {
  it("shows every reviewed change's preview, waits for them, and sends each digest", async () => {
    listed = [
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: false }),
      proposal({ id: "aprop_3", reviewRequired: true }),
    ];
    let release: (() => void) | null = null;
    fetchProposalPreview.mockImplementation(
      (request) =>
        new Promise((resolve) => {
          const answer = () => resolve(preview({ digest: `sha256:${request.id}` }));
          if (request.id === "aprop_3") {
            release = answer;
          } else {
            answer();
          }
        }),
    );
    decideMyProposals.mockResolvedValue([]);
    renderBody(["aprop_1", "aprop_2", "aprop_3"]);

    await screen.findByText("Waiting for 1 preview to load");
    expect(approve()).toBeDisabled();
    expect(fetchProposalPreview.mock.calls.map(([request]) => request.id).sort()).toEqual([
      "aprop_1",
      "aprop_3",
    ]);

    release!();
    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());

    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2", "aprop_3"], {
      decision: "Accepted",
      reasonCode: "",
      previewDigests: [
        { proposalId: "aprop_1", digest: "sha256:aprop_1" },
        { proposalId: "aprop_3", digest: "sha256:aprop_3" },
      ],
    });
  });

  it("never counts a preview of a record that moved since as reviewed", async () => {
    listed = [
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: true }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ digest: `sha256:${request.id}`, stale: request.id === "aprop_2" }),
    );
    renderBody(["aprop_1", "aprop_2"]);

    await screen.findByText("Waiting for 1 preview to load");
    expect(approve()).toBeDisabled();
  });

  it("approves internal changes together without previews", async () => {
    listed = [
      proposal({ id: "aprop_1", reviewRequired: false }),
      proposal({ id: "aprop_2", reviewRequired: false }),
    ];
    decideMyProposals.mockResolvedValue([]);
    renderBody(["aprop_1", "aprop_2"]);

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(fetchProposalPreview).not.toHaveBeenCalled();
    expect(decideMyProposals).toHaveBeenCalledWith(
      ["aprop_1", "aprop_2"],
      expect.objectContaining({ previewDigests: [] }),
    );
  });
});
