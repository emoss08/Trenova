import { planPreview, preview } from "@/components/assistant/__tests__/preview-fixtures";
import type {
  PlanPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import type { AssistantArtifact, AssistantPlan, AssistantProposal } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from "vitest";
import { toast } from "@trenova/shared/components/ui/toast";
import { DeskDecisionBody } from "../desk-decision-body";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchPlanPreview =
  vi.fn<(request: { scope: string; id: string }, options?: unknown) => Promise<PlanPreview>>();
const decideMyProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideMyPlan = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let listed: AssistantProposal[] = [];
let listedPlans: AssistantPlan[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchPlanPreview: (request: { scope: string; id: string }, options?: unknown) =>
    fetchPlanPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: vi.fn(),
  decideMyProposals: (...args: unknown[]) => decideMyProposals(...args),
  decideMyPlan: (...args: unknown[]) => decideMyPlan(...args),
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/lib/queries/assistant")>();
  return {
    ...original,
    assistant: {
      ...original.assistant,
      // The cache helpers invalidate a factory by its `_def` scope, so the
      // stand-ins carry one that prefixes the keys they hand out.
      proposals: Object.assign(
        (threadId: string) => ({
          queryKey: ["assistant-proposals", threadId],
          queryFn: async () => ({ results: listed }),
        }),
        { _def: ["assistant-proposals"] },
      ),
      plans: Object.assign(
        (threadId: string) => ({
          queryKey: ["assistant-plans", threadId],
          queryFn: async () => ({ results: listedPlans }),
        }),
        { _def: ["assistant-plans"] },
      ),
    },
  };
});

afterEach(() => {
  cleanup();
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
  decideMyProposals.mockReset();
  decideMyPlan.mockReset();
  listed = [];
  listedPlans = [];
});

/* services.AssistantProposal as the thread's proposals route serves it. */
function proposal(overrides: Partial<AssistantProposal>): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "receivable_post_customer_payment",
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

function renderBody(ids: string[], planId = "") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const artifact = {
    id: "aart_1",
    threadId: "athr_1",
    kind: "decision_request",
    title: "Post these payments",
    proposalId: ids[0],
    payload: planId !== "" ? { planId } : { proposalIds: ids },
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

    await screen.findByText("1 change has changed since it was drafted and can only be set aside");
    expect(screen.queryByText(/Waiting for/)).toBeNull();
    expect(approve()).toBeDisabled();
  });

  it("says a preview failed to load rather than that it is still loading", async () => {
    listed = [
      proposal({ id: "aprop_1", reviewRequired: true }),
      proposal({ id: "aprop_2", reviewRequired: true }),
    ];
    fetchProposalPreview.mockImplementation(async (request) => {
      if (request.id === "aprop_2") {
        throw new Error("preview unavailable");
      }
      return preview({ digest: `sha256:${request.id}` });
    });
    renderBody(["aprop_1", "aprop_2"]);

    await screen.findByText("1 preview could not be loaded; try it again above");
    expect(
      screen.getByText(
        "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
      ),
    ).toBeInTheDocument();
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

/* services.AssistantPlan as the thread's plans route serves it. */
function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Post two payments",
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

describe("DeskDecisionBody approving a plan", () => {
  it("shows the plan's preview when a step must be reviewed and sends its digest", async () => {
    listedPlans = [plan()];
    listed = [
      proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true }),
      proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: false }),
    ];
    fetchPlanPreview.mockResolvedValue(
      planPreview({ digest: "sha256:plan", reviewRequired: true }),
    );
    decideMyPlan.mockResolvedValue({
      refused: false,
      plan: { id: "apl_1", status: "Approving", commitsAt: 1_030, failureError: "" },
    });
    renderBody([], "apl_1");

    await waitFor(() => expect(approve()).toBeEnabled());
    expect(fetchPlanPreview).toHaveBeenCalledWith(
      expect.objectContaining({ scope: "mine", id: "apl_1" }),
      expect.anything(),
    );
    fireEvent.click(approve());

    await waitFor(() => expect(decideMyPlan).toHaveBeenCalledTimes(1));
    expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Accepted",
      reasonCode: "",
      previewDigest: "sha256:plan",
    });
  });

  it("keeps a plan with a step that must be reviewed off until its preview is current", async () => {
    listedPlans = [plan()];
    listed = [proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true })];
    fetchPlanPreview.mockResolvedValue(planPreview({ stale: true, reviewRequired: true }));
    renderBody([], "apl_1");

    await screen.findByText(
      "The plan's records changed since it was drafted, so it can only be set aside",
    );
    expect(approve()).toBeDisabled();
    expect(decideMyPlan).not.toHaveBeenCalled();
  });
});

/**
 * A plan approval the server refuses because a record a step changes moved on
 * since it was proposed is recorded, runs nothing, and comes back `refused`
 * with the plan failed at that step. The card says at once that nothing
 * changed, and reads as not carried out rather than approved.
 */
describe("DeskDecisionBody when the server refuses a plan approval", () => {
  const why =
    "nothing ran, because a record the plan changes moved on after it was proposed; step 1: the record changed since this was proposed: the customer payment CP-1 was at version 2 when this was proposed and is at version 3 now, so nothing was changed";
  let info: MockInstance;

  beforeEach(() => {
    info = vi.spyOn(toast, "info");
  });

  afterEach(() => {
    info.mockRestore();
  });

  it("says nothing changed and reads as not carried out", async () => {
    listedPlans = [plan()];
    listed = [
      proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: false }),
      proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: false }),
    ];
    fetchPlanPreview.mockResolvedValue(planPreview({ digest: "sha256:plan" }));
    decideMyPlan.mockImplementation(async () => {
      listedPlans = [plan({ status: "Failed", failedStep: 1, failureError: why })];
      return {
        refused: true,
        plan: {
          id: "apl_1",
          status: "Failed",
          stepCount: 2,
          completedSteps: 0,
          failedStep: 1,
          failureError: why,
          decidedByUserId: "usr_1",
          decidedAt: 1_000,
          commitsAt: null,
          version: 2,
          updatedAt: 1_000,
        },
      };
    });
    renderBody([], "apl_1");

    await waitFor(() => expect(approve()).toBeEnabled());
    fireEvent.click(approve());

    await waitFor(() => expect(info).toHaveBeenCalledTimes(1));
    expect(info).toHaveBeenCalledWith("This changed since you opened it", { description: why });
    expect(await screen.findByText("Not carried out")).toBeInTheDocument();
    expect(screen.queryByText("Approved")).toBeNull();
  });
});

describe("DeskDecisionBody showing a script's batch", () => {
  it("says a script wrote the batch, what it was for, and shows its code on request", async () => {
    listedPlans = [
      plan({
        script: {
          title: "Hold Range's uncovered loads",
          purpose: "Hold every Range load out of Denver with nobody on it.",
          code: "for load in loads:\n    trenova.shipment.place_hold(shipmentId=load)",
        },
      }),
    ];
    listed = [
      proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: true }),
      proposal({ id: "aprop_2", planId: "apl_1", planStep: 2, reviewRequired: true }),
    ];
    fetchPlanPreview.mockResolvedValue(
      planPreview({ digest: "sha256:plan", reviewRequired: true }),
    );
    renderBody([], "apl_1");

    expect(
      await screen.findByText("Written by a script: Hold Range's uncovered loads"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Hold every Range load out of Denver with nobody on it."),
    ).toBeInTheDocument();
    expect(screen.queryByText(/place_hold/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Show the script" }));

    expect(await screen.findByText(/place_hold/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Hide the script" })).toBeInTheDocument();
  });

  it("names no script for a plan of the model's own calls", async () => {
    listedPlans = [plan()];
    listed = [proposal({ id: "aprop_1", planId: "apl_1", planStep: 1, reviewRequired: false })];
    renderBody([], "apl_1");

    expect(await screen.findByText("Post these payments")).toBeInTheDocument();
    expect(screen.queryByText(/Written by a script/)).not.toBeInTheDocument();
  });
});
