import { approvalQueue, type ApprovalEntry } from "@/components/assistant/approval-queue";
import { planPreview, preview, record } from "@/components/assistant/__tests__/preview-fixtures";
import type {
  PlanPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import type { AssistantPlan, AssistantProposal } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskApprovalCard } from "../desk-approval-card";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchPlanPreview =
  vi.fn<(request: { scope: string; id: string }, options?: unknown) => Promise<PlanPreview>>();
const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideMyProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideMyPlan = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchPlanPreview: (request: { scope: string; id: string }, options?: unknown) =>
    fetchPlanPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
  decideMyProposals: (...args: unknown[]) => decideMyProposals(...args),
  decideMyPlan: (...args: unknown[]) => decideMyPlan(...args),
  decideAgentProposal: vi.fn(),
}));

beforeEach(() => {
  decideMyProposal.mockResolvedValue({ commitsAt: null });
});

afterEach(() => {
  cleanup();
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
  decideMyProposal.mockReset();
  decideMyProposals.mockReset();
  decideMyPlan.mockReset();
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
    reviewRequired: true,
    ...overrides,
  };
}

/* services.AssistantPlan as the thread's routes serve it. */
function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Reprice two loads",
    summary: "",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    expiresAt: 0,
    hold: null,
    createdAt: 100,
    agentId: null,
    agentName: null,
    ...overrides,
  };
}

function entryOf(proposals: AssistantProposal[], plans: AssistantPlan[] = []): ApprovalEntry {
  const [first] = approvalQueue(proposals, plans, null, 1_000);
  if (!first) {
    throw new Error("nothing waiting");
  }
  return first;
}

function renderCard(proposals: AssistantProposal[] = [proposal()], plans: AssistantPlan[] = []) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const approveRef = { current: null as (() => void) | null };
  const undo = { start: vi.fn(), scheduled: vi.fn(), clear: vi.fn() };
  const onDecided = vi.fn();
  const onReview = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DeskApprovalCard
          threadId="athr_1"
          entry={entryOf(proposals, plans)}
          approveRef={approveRef}
          onReview={onReview}
          onDefer={vi.fn()}
          onDecided={onDecided}
          undo={undo}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { approveRef, undo, onDecided, onReview };
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

/**
 * A new shipment's preview lists the shipment and one change per stop. The
 * card counted every change as a record of the first one's kind, so a load
 * with four stops read "Enter a shipment on 5 shipments"; and a record being
 * created is not one the change acts "on".
 */
describe("DeskApprovalCard for a change that creates records", () => {
  const stop = (sequence: number) =>
    record({
      resource: "shipment_stop",
      record: null,
      entityId: null,
      label: `Pickup stop ${sequence}`,
      operation: "Create",
    });

  it("counts the records it creates, not their parts, and acts on none", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({
        tool: "create_shipment",
        changes: [
          record({ record: null, entityId: null, label: "New shipment", operation: "Create" }),
          stop(1),
          stop(2),
          stop(3),
          stop(4),
        ],
      }),
    );
    renderCard();

    await waitFor(() => expect(approve()).toBeEnabled());
    expect(screen.queryByText(/shipments/)).toBeNull();
    expect(screen.queryByText(/^on /)).toBeNull();
  });
});

/**
 * The server approves a change that moves money, reaches a customer, a driver
 * or someone outside, or was proposed after outside text was read, only with
 * the digest of a preview the person was shown (services.AssistantProposal
 * reviewRequired). Internal and personal changes may still go unreviewed.
 */
describe("DeskApprovalCard and changes that must be reviewed", () => {
  it("keeps Approve off when a reviewed change's preview cannot be read", async () => {
    fetchProposalPreview.mockRejectedValue(new Error("preview service down"));
    const { approveRef } = renderCard([proposal({ reviewRequired: true })]);

    expect(
      await screen.findByText(
        "Its preview could not be read, and this change is approved only once you've seen it.",
      ),
    ).toBeInTheDocument();
    expect(approve()).toBeDisabled();
    expect(approveRef.current).toBeNull();
  });

  it("still approves an internal change unreviewed when its preview cannot be read", async () => {
    fetchProposalPreview.mockRejectedValue(new Error("preview service down"));
    renderCard([proposal({ reviewRequired: false })]);

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposal).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).toHaveBeenCalledWith(
      "aprop_1",
      expect.objectContaining({ decision: "Accepted", previewDigest: undefined }),
    );
  });

  it("approves edited wording with the digest of the preview of those edits", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:edited" }));
    renderCard([proposal({ reviewRequired: true, pendingModifications: { status: "AtRisk" } })]);

    await waitFor(() => expect(approve()).toBeEnabled());
    expect(fetchProposalPreview).toHaveBeenCalledWith(
      expect.objectContaining({ id: "aprop_1", modifications: { status: "AtRisk" } }),
      expect.anything(),
    );
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposal).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
      decision: "Modified",
      reasonCode: "modified_from_desk",
      modifications: { status: "AtRisk" },
      previewDigest: "sha256:edited",
    });
  });

  it("opens each change of a batch that must be reviewed and approves with their digests", async () => {
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, digest: `sha256:${request.id}` }),
    );
    decideMyProposals.mockResolvedValue([
      { proposalId: "aprop_1", decision: { commitsAt: null }, error: null },
      { proposalId: "aprop_2", decision: { commitsAt: null }, error: null },
      { proposalId: "aprop_3", decision: { commitsAt: null }, error: null },
    ]);
    const { onReview, approveRef } = renderCard([
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: false, arguments: { shipmentId: "shp_2" } }),
      proposal({ id: "aprop_3", reviewRequired: true, arguments: { shipmentId: "shp_3" } }),
    ]);

    const review = await screen.findByRole("button", { name: "Review 2" });
    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    expect(approveRef.current).toBeNull();
    fireEvent.click(review);
    expect(onReview).not.toHaveBeenCalled();

    const reviewed = await screen.findByRole("group", {
      name: "Changes to review before approving",
    });
    expect(within(reviewed).getAllByRole("region", { name: /^Preview of / })).toHaveLength(2);
    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());

    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals).toHaveBeenCalledWith(
      ["aprop_1", "aprop_2", "aprop_3"],
      expect.objectContaining({
        decision: "Accepted",
        previewDigests: [
          { proposalId: "aprop_1", digest: "sha256:aprop_1" },
          { proposalId: "aprop_3", digest: "sha256:aprop_3" },
        ],
      }),
    );
  });

  it("says why a reviewed batch still cannot be approved", async () => {
    fetchProposalPreview.mockImplementation(async (request) => {
      if (request.id === "aprop_2") {
        throw new Error("preview unavailable");
      }
      return preview({
        proposalId: request.id,
        digest: `sha256:${request.id}`,
        stale: request.id === "aprop_3",
      });
    });
    renderCard([
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: true, arguments: { shipmentId: "shp_2" } }),
      proposal({ id: "aprop_3", reviewRequired: true, arguments: { shipmentId: "shp_3" } }),
    ]);

    fireEvent.click(await screen.findByRole("button", { name: "Review 3" }));

    expect(
      await screen.findByText(
        "1 change has changed since it was drafted and can only be set aside · 1 preview could not be loaded; try it again above",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    expect(decideMyProposals).not.toHaveBeenCalled();
  });

  it("leaves a change that moved out of the batch and approves the rest", async () => {
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        digest: `sha256:${request.id}`,
        stale: request.id === "aprop_2",
      }),
    );
    decideMyProposals.mockResolvedValue([
      { proposalId: "aprop_1", decision: { commitsAt: null }, error: null },
    ]);
    renderCard([
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: true, arguments: { shipmentId: "shp_2" } }),
    ]);

    fireEvent.click(await screen.findByRole("button", { name: "Review 2" }));

    expect(
      await screen.findByText(/1 left out: it changed since it was drafted/),
    ).toBeInTheDocument();
    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals).toHaveBeenCalledWith(
      ["aprop_1"],
      expect.objectContaining({
        previewDigests: [{ proposalId: "aprop_1", digest: "sha256:aprop_1" }],
      }),
    );
  });

  it("opens the decision from a batch whose review is already open", async () => {
    fetchProposalPreview.mockImplementation(async (request) => {
      if (request.id === "aprop_2") {
        throw new Error("preview unavailable");
      }
      return preview({ proposalId: request.id, digest: `sha256:${request.id}` });
    });
    const { onReview } = renderCard([
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: true, arguments: { shipmentId: "shp_2" } }),
    ]);

    fireEvent.click(await screen.findByRole("button", { name: "Review 2" }));
    expect(onReview).not.toHaveBeenCalled();
    const open = await screen.findByRole("button", { name: "Open the decision" });
    expect(open).toBeEnabled();
    fireEvent.click(open);
    expect(onReview).toHaveBeenCalledTimes(1);
  });

  it("never batches a change the person edited with others", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:edited" }));
    renderCard([
      proposal({ id: "aprop_1", reviewRequired: false, pendingModifications: { status: "Late" } }),
      proposal({ id: "aprop_2", reviewRequired: false, arguments: { shipmentId: "shp_2" } }),
    ]);

    await waitFor(() => expect(approve()).toBeEnabled());
    expect(screen.queryByRole("button", { name: /^Review \d/ })).toBeNull();
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposal).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).toHaveBeenCalledWith(
      "aprop_1",
      expect.objectContaining({ decision: "Modified", modifications: { status: "Late" } }),
    );
    expect(decideMyProposals).not.toHaveBeenCalled();
  });

  it("holds an edited change for its preview even when the change as proposed needs none", async () => {
    fetchProposalPreview.mockRejectedValue(new Error("preview unavailable"));
    renderCard([
      proposal({
        reviewRequired: false,
        pendingModifications: { status: "Cancelled" },
      }),
    ]);

    await waitFor(() => expect(fetchProposalPreview).toHaveBeenCalled());
    await act(async () => {});
    expect(fetchProposalPreview.mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({ modifications: { status: "Cancelled" } }),
    );
    expect(approve()).toBeDisabled();
  });

  it("approves an edit that its preview says needs no review even when the proposal did", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({ digest: "sha256:edited", reviewRequired: false }),
    );
    renderCard([proposal({ reviewRequired: true, pendingModifications: { status: "Delayed" } })]);

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposal).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).toHaveBeenCalledWith(
      "aprop_1",
      expect.objectContaining({ decision: "Modified", previewDigest: "sha256:edited" }),
    );
  });

  it("keeps a plan with a step that must be reviewed off until its preview is in", async () => {
    let release: ((value: PlanPreview) => void) | null = null;
    fetchPlanPreview.mockImplementation(
      () =>
        new Promise((resolve) => {
          release = resolve;
        }),
    );
    decideMyPlan.mockResolvedValue({ commitsAt: null });
    renderCard(
      [
        proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true }),
        proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: false }),
      ],
      [plan()],
    );

    expect(approve()).toBeDisabled();
    await waitFor(() => expect(release).not.toBeNull());
    await act(async () => release!(planPreview({ digest: "sha256:plan", reviewRequired: true })));
    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyPlan).toHaveBeenCalledTimes(1));
    expect(decideMyPlan).toHaveBeenCalledWith(
      "apl_1",
      expect.objectContaining({ previewDigest: "sha256:plan" }),
    );
  });

  it("holds a plan whose steps are not listed yet when its preview cannot be read", async () => {
    fetchPlanPreview.mockRejectedValue(new Error("preview unavailable"));
    renderCard([], [plan()]);

    await waitFor(() => expect(fetchPlanPreview).toHaveBeenCalled());
    await act(async () => {});
    expect(approve()).toBeDisabled();
  });

  it("keeps a plan with a step that must be reviewed off when its preview cannot be read", async () => {
    fetchPlanPreview.mockRejectedValue(new Error("preview unavailable"));
    renderCard(
      [
        proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true }),
        proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: false }),
      ],
      [plan()],
    );

    await waitFor(() => expect(fetchPlanPreview).toHaveBeenCalled());
    await act(async () => {});
    expect(approve()).toBeDisabled();
    expect(decideMyPlan).not.toHaveBeenCalled();
  });

  it("approves a batch of internal changes from the card", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:first" }));
    decideMyProposals.mockResolvedValue([
      { proposalId: "aprop_1", decision: { commitsAt: null }, error: null },
      { proposalId: "aprop_2", decision: { commitsAt: null }, error: null },
    ]);
    renderCard([
      proposal({ id: "aprop_1", reviewRequired: false }),
      proposal({ id: "aprop_2", reviewRequired: false, arguments: { shipmentId: "shp_2" } }),
    ]);

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());
    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals).toHaveBeenCalledWith(
      ["aprop_1", "aprop_2"],
      expect.objectContaining({ decision: "Accepted", previewDigests: [] }),
    );
  });
});
