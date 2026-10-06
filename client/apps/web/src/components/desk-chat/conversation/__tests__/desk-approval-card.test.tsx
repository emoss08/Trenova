import { approvalQueue, type ApprovalEntry } from "@/components/assistant/approval-queue";
import { preview } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import type { AssistantProposal } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskApprovalCard } from "../desk-approval-card";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
  decideMyProposals: vi.fn(),
  decideMyPlan: vi.fn(),
  decideAgentProposal: vi.fn(),
}));

beforeEach(() => {
  decideMyProposal.mockResolvedValue({ commitsAt: null });
});

afterEach(() => {
  cleanup();
  fetchProposalPreview.mockReset();
  decideMyProposal.mockReset();
});

/* services.AssistantProposal as the thread's routes serve it; outside any plan. */
function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "update_shipment",
    arguments: { shipmentId: "shp_1", status: "Delayed" },
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
    createdAt: 100,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    ...overrides,
  };
}

function entryOf(proposals: AssistantProposal[]): ApprovalEntry {
  const [first] = approvalQueue(proposals, [], null, 1_000);
  if (!first) {
    throw new Error("nothing waiting");
  }
  return first;
}

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const approveRef = { current: null as (() => void) | null };
  const undo = { start: vi.fn(), scheduled: vi.fn(), clear: vi.fn() };
  const onDecided = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DeskApprovalCard
          threadId="athr_1"
          entry={entryOf([proposal()])}
          approveRef={approveRef}
          onReview={vi.fn()}
          onDefer={vi.fn()}
          onDecided={onDecided}
          undo={undo}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { approveRef, undo, onDecided };
}

const approve = () => screen.getByRole("button", { name: /^Approve/ });

/**
 * Ported from the assistant's approval box: the one approval card both
 * surfaces now share approves only what the person was shown, and only once
 * they have been shown it.
 */
describe("DeskApprovalCard for one proposal", () => {
  it("reads the preview as the conversation's owner and keeps Approve off until it is in", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    const { approveRef } = renderCard();

    expect(fetchProposalPreview).toHaveBeenCalledWith(
      expect.objectContaining({ scope: "mine", id: "aprop_1" }),
      expect.anything(),
    );
    expect(approve()).toBeDisabled();
    // ⌘↵ has nothing to approve yet.
    expect(approveRef.current).toBeNull();
  });

  it("approves with the digest of the preview on screen, and opens the undo window", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    const { undo, onDecided } = renderCard();

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:shown",
      }),
    );
    expect(undo.start).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(onDecided).toHaveBeenCalledTimes(1));
  });

  it("hands ⌘↵ the same approval once the preview is in", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    const { approveRef } = renderCard();

    await waitFor(() => expect(approveRef.current).not.toBeNull());
    await act(async () => approveRef.current?.());

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Accepted", previewDigest: "sha256:shown" }),
      ),
    );
  });

  it("offers no approval for a record that changed since it was proposed", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ stale: true, digest: "sha256:stale" }));
    const { approveRef } = renderCard();

    await screen.findByRole("group", { name: "Out of date" });
    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    expect(approveRef.current).toBeNull();
    expect(decideMyProposal).not.toHaveBeenCalled();
  });
});
