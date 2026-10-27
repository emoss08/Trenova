import {
  field,
  message,
  planPreview,
  preview,
  record,
} from "@/components/assistant/__tests__/preview-fixtures";
import type { PlanPreview } from "@/lib/graphql/agent-preview";
import { useDeskStore } from "@/stores/desk-store";
import type {
  AssistantArtifact,
  AssistantPlan,
  AssistantProposal,
  AssistantThread,
} from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskPlanBody } from "../desk-plan-body";

const NOW = 1_790_000_000;
const THREAD = "athr_1";
const fetchPlanPreview =
  vi.fn<(request: { scope: string; id: string }, options?: unknown) => Promise<PlanPreview>>();
const decideMyPlan = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const undoMyDecision = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let listedPlans: AssistantPlan[] = [];
let listedSteps: AssistantProposal[] = [];
let thread: AssistantThread | null = null;

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchPlanPreview: (request: { scope: string; id: string }, options?: unknown) =>
    fetchPlanPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyPlan: (...args: unknown[]) => decideMyPlan(...args),
  undoMyDecision: (...args: unknown[]) => undoMyDecision(...args),
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/lib/queries/assistant")>();
  return {
    ...original,
    assistant: {
      ...original.assistant,
      proposals: (threadId: string) => ({
        queryKey: ["assistant-proposals", threadId],
        queryFn: async () => ({ results: listedSteps }),
      }),
      plans: (threadId: string) => ({
        queryKey: ["assistant-plans", threadId],
        queryFn: async () => ({ results: listedPlans }),
      }),
      thread: (id: string) => ({
        queryKey: ["assistant-thread", id],
        queryFn: async () => thread,
      }),
    },
  };
});

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now: NOW * 1000 });
  useAuthStore.setState({
    user: { id: "usr_me" } as ReturnType<typeof useAuthStore.getState>["user"],
  });
  useDeskStore.setState({ asks: {} });
  thread = ownThread();
  fetchPlanPreview.mockResolvedValue(fixturePreview());
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  fetchPlanPreview.mockReset();
  decideMyPlan.mockReset();
  undoMyDecision.mockReset();
  listedPlans = [];
  listedSteps = [];
});

/* services.AssistantThread as the thread route serves it. */
function ownThread(overrides: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id: THREAD,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    userId: "usr_me",
    agentDefinitionId: "agdef_1",
    title: "",
    status: "Active",
    lastMessageAt: NOW,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    createdAt: NOW - 3600,
    updatedAt: NOW,
    ...overrides,
  } as AssistantThread;
}

/* services.AssistantPlan as the thread's plans route serves it. */
function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Recover S-1001 after the dock delay",
    summary: "The dock turned the truck away.",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    decidedByName: "",
    decisionNote: "",
    previewDigest: "",
    commitsAt: null,
    undoneAt: null,
    runMode: "InOrder",
    expiresAt: NOW + 86400,
    hold: null,
    createdAt: NOW - 600,
    agentId: "agdef_1",
    agentName: "Dispatch",
    ...overrides,
  };
}

/* services.AssistantProposal as the thread's proposals route serves it. */
function step(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "stop_update_appointment",
    arguments: {},
    rationale: "Move the delivery to tomorrow",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: NOW + 86400,
    planId: "apl_1",
    planStep: 1,
    fields: [],
    createdAt: NOW - 600,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    reviewRequired: true,
    ...overrides,
  };
}

function steps(statuses: [AssistantProposal["status"], AssistantProposal["status"]]) {
  return [
    step({
      status: statuses[0],
      executedAt: statuses[0] === "Executed" ? NOW - 300 : null,
    }),
    step({
      id: "aprop_2",
      planStep: 2,
      toolName: "email_send",
      rationale: "Tell the customer",
      status: statuses[1],
      executedAt:
        statuses[1] === "Executed" || statuses[1] === "ExecutionFailed" ? NOW - 290 : null,
      executionError: statuses[1] === "ExecutionFailed" ? "The customer has no email on file." : "",
    }),
  ];
}

function fixturePreview(): PlanPreview {
  return planPreview({
    digest: "sha256:plan-digest",
    steps: [
      {
        proposalId: "aprop_1",
        step: 1,
        preview: preview({
          proposalId: "aprop_1",
          changes: [
            record({
              label: "S-1001",
              fields: [
                field({
                  path: "appointment",
                  label: "Stop 2 appointment",
                  before: "Oct 20, 2:00 PM",
                  after: "Oct 21, 8:00 AM",
                }),
              ],
            }),
          ],
        }),
      },
      {
        proposalId: "aprop_2",
        step: 2,
        preview: preview({
          proposalId: "aprop_2",
          changes: [
            record({
              resource: "customer",
              record: { entityType: "customer", id: "cus_1" },
              entityId: "cus_1",
              label: "Tyson Foods",
              operation: "Send",
              fields: [],
              message: message({ subject: "Delivery moved" }),
            }),
          ],
        }),
      },
    ],
  });
}

function renderPlan() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const artifact = {
    id: "aart_1",
    threadId: THREAD,
    planId: "apl_1",
    proposalId: "",
    kind: "plan",
    title: "Recover S-1001 after the dock delay",
    payload: {
      title: "Recover S-1001 after the dock delay",
      summary: "",
      stepCount: 2,
      steps: [
        { step: 1, proposalId: "aprop_1", toolName: "stop_update_appointment", rationale: "" },
        { step: 2, proposalId: "aprop_2", toolName: "email_send", rationale: "" },
      ],
    },
  } as unknown as AssistantArtifact;
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DeskPlanBody artifact={artifact} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
}

const approveButton = () => screen.queryByRole("button", { name: "Approve plan" });
const declineButton = () => screen.queryByRole("button", { name: "Decline" });
const finishButton = () => screen.queryByRole("button", { name: "Ask the agent to finish it" });

describe("DeskPlanBody while the plan waits", () => {
  it("draws each step's change from what to what, the records it touches, and asks the owner", async () => {
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    renderPlan();

    await screen.findByText("Awaiting your approval");
    expect(screen.getByText("Recover S-1001 after the dock delay")).toBeInTheDocument();
    expect(screen.getByText("Move the delivery to tomorrow")).toBeInTheDocument();
    expect(await screen.findByText("Oct 20, 2:00 PM")).toBeInTheDocument();
    expect(screen.getByText("Oct 21, 8:00 AM")).toBeInTheDocument();
    expect(screen.getByText("Delivery moved")).toBeInTheDocument();
    expect(screen.getByText("Records it touches")).toBeInTheDocument();
    expect(screen.getByText("2 changes")).toBeInTheDocument();
    expect(screen.getByText("on 2 records")).toBeInTheDocument();
    expect(screen.getByText("Proposed by Dispatch")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Changed/ })).toBeNull();
    expect(fetchPlanPreview).toHaveBeenCalledWith(
      { scope: "mine", id: "apl_1" },
      expect.anything(),
    );
    await waitFor(() => expect(approveButton()).toBeEnabled());
    expect(declineButton()).toBeEnabled();
  });

  it("approves against the digest of the preview it shows, with no toast", async () => {
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    decideMyPlan.mockResolvedValue({
      refused: false,
      plan: { id: "apl_1", status: "Approving", commitsAt: NOW + 10, failureError: "" },
    });
    renderPlan();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    fireEvent.click(approveButton()!);

    await waitFor(() => expect(decideMyPlan).toHaveBeenCalledTimes(1));
    expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Accepted",
      reasonCode: "approved_from_plan_card",
      previewDigest: "sha256:plan-digest",
    });
  });

  it("reads the preview again and says so when the server refuses a digest that moved", async () => {
    const { GraphQLRequestError } = await import("@trenova/shared/lib/graphql");
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    decideMyPlan.mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        message: "What this plan would do has changed since you reviewed it",
        status: 200,
        graphQLErrors: [
          {
            message: "What this plan would do has changed since you reviewed it",
            extensions: { type: "https://trenova.app/problems/resource-conflict" },
            type: "https://trenova.app/problems/resource-conflict",
          },
        ],
      }),
    );
    renderPlan();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    fireEvent.click(approveButton()!);

    expect(
      await screen.findByText("This change looks different now — review it again."),
    ).toBeInTheDocument();
    await waitFor(() => expect(fetchPlanPreview.mock.calls.length).toBeGreaterThan(1));
  });

  it("declines with the note the person wrote, which the agent receives", async () => {
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    decideMyPlan.mockResolvedValue({
      refused: false,
      plan: { id: "apl_1", status: "Rejected", commitsAt: null, failureError: "" },
    });
    renderPlan();

    await waitFor(() => expect(declineButton()).toBeEnabled());
    fireEvent.click(declineButton()!);
    const dialog = await screen.findByRole("dialog");
    fireEvent.change(within(dialog).getByRole("textbox"), {
      target: { value: "  I'm calling them myself.  " },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Decline plan" }));

    await waitFor(() => expect(decideMyPlan).toHaveBeenCalledTimes(1));
    expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Rejected",
      reasonCode: "declined_from_plan_card",
      note: "I'm calling them myself.",
    });
  });

  it("declines without a note when the person leaves it empty", async () => {
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    decideMyPlan.mockResolvedValue({
      refused: false,
      plan: { id: "apl_1", status: "Rejected", commitsAt: null, failureError: "" },
    });
    renderPlan();

    await waitFor(() => expect(declineButton()).toBeEnabled());
    fireEvent.click(declineButton()!);
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Decline plan" }));

    await waitFor(() => expect(decideMyPlan).toHaveBeenCalledTimes(1));
    expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
      decision: "Rejected",
      reasonCode: "declined_from_plan_card",
      note: undefined,
    });
  });

  it("offers no decision to someone who does not own the conversation", async () => {
    thread = ownThread({ userId: "usr_someone_else" });
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    renderPlan();

    await screen.findByText("Awaiting approval");
    expect(approveButton()).toBeNull();
    expect(declineButton()).toBeNull();
    expect(
      screen.getByText("It is decided in the conversation it was proposed in."),
    ).toBeInTheDocument();
  });

  it("offers no decision when the agent can no longer be asked", async () => {
    thread = ownThread({ canContinue: false });
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    renderPlan();

    await screen.findByText("Awaiting approval");
    expect(approveButton()).toBeNull();
  });

  it("holds a plan a shadow switch keeps from being decided, with no buttons", async () => {
    listedPlans = [plan({ hold: { reason: "AgentShadow", agentName: "Dispatch" } })];
    listedSteps = steps(["Pending", "Pending"]);
    renderPlan();

    await screen.findByText("On hold");
    expect(screen.getByText("This plan is on hold")).toBeInTheDocument();
    expect(screen.getByText("Dispatch is in shadow")).toBeInTheDocument();
    expect(approveButton()).toBeNull();
  });
});

describe("DeskPlanBody once approved", () => {
  it("counts down the undo window and undoes the approval", async () => {
    listedPlans = [
      plan({
        status: "Approving",
        decidedAt: NOW - 2,
        decidedByUserId: "usr_me",
        decidedByName: "Maya Chen",
        previewDigest: "sha256:plan-digest",
        commitsAt: NOW + 8,
      }),
    ];
    listedSteps = steps(["Approving", "Approving"]);
    undoMyDecision.mockResolvedValue(true);
    renderPlan();

    await screen.findByText("Approved");
    expect(screen.getByText("Can be undone for 0:08")).toBeInTheDocument();
    expect(await screen.findByText("Oct 21, 8:00 AM")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(undoMyDecision).toHaveBeenCalledWith({ planId: "apl_1" }));
  });

  it("lets only the person who approved it undo it", async () => {
    listedPlans = [
      plan({
        status: "Approving",
        decidedByUserId: "usr_other",
        decidedByName: "Ana Ruiz",
        commitsAt: NOW + 8,
      }),
    ];
    listedSteps = steps(["Approving", "Approving"]);
    renderPlan();

    await screen.findByText("Approved");
    expect(screen.queryByRole("button", { name: "Undo" })).toBeNull();
    expect(screen.getByText("Approved by Ana Ruiz")).toBeInTheDocument();
  });

  it("shows the step running, the one before it ran and folded, and no actions", async () => {
    listedPlans = [
      plan({
        status: "Approved",
        decidedByUserId: "usr_me",
        decidedAt: NOW - 400,
        previewDigest: "sha256:plan-digest",
      }),
    ];
    listedSteps = [
      step({ status: "Executed", executedAt: NOW - 300 }),
      step({ id: "aprop_2", planStep: 2, status: "Accepted", toolName: "email_send" }),
    ];
    renderPlan();

    await screen.findByText("Running step 2");
    expect(screen.getByText("1 of 2")).toBeInTheDocument();
    expect(screen.getByText("Running…")).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: /Changed 1 field on S-1001/ }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Oct 20, 2:00 PM")).toBeNull();
    expect(screen.getByText("Approved against plan-digest")).toBeInTheDocument();
    expect(approveButton()).toBeNull();
    expect(finishButton()).toBeNull();
  });

  it("opens a folded step's changes on request", async () => {
    listedPlans = [plan({ status: "Completed", decidedByUserId: "usr_me" })];
    listedSteps = steps(["Executed", "Executed"]);
    renderPlan();

    const fold = await screen.findByRole("button", { name: /Changed 1 field on S-1001/ });
    expect(fold).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(fold);
    expect(fold).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Oct 20, 2:00 PM")).toBeInTheDocument();
  });

  it("keeps its own words when a later plan follows a plan that finished", async () => {
    listedPlans = [
      plan({ status: "Completed", decidedByUserId: "usr_me" }),
      plan({ id: "apl_2", title: "Something else", createdAt: NOW - 60 }),
    ];
    listedSteps = steps(["Executed", "Executed"]);
    renderPlan();

    await screen.findByText("Done");
    expect(screen.getByText("Ran as approved.")).toBeInTheDocument();
    expect(screen.queryByText("A later plan in this conversation takes it from here.")).toBeNull();
  });

  it("folds a step that was open while the plan waited once the plan has run", async () => {
    listedPlans = [plan()];
    listedSteps = steps(["Pending", "Pending"]);
    const client = renderPlan();

    expect(await screen.findByText("Oct 20, 2:00 PM")).toBeInTheDocument();
    listedPlans = [plan({ status: "Completed", decidedByUserId: "usr_me" })];
    listedSteps = steps(["Executed", "Executed"]);
    await client.invalidateQueries();

    const fold = await screen.findByRole("button", { name: /Changed 1 field on S-1001/ });
    expect(fold).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Oct 20, 2:00 PM")).toBeNull();
  });

  it("calls a plan whose every step ran done", async () => {
    listedPlans = [plan({ status: "Completed", decidedByUserId: "usr_me" })];
    listedSteps = steps(["Executed", "Executed"]);
    renderPlan();

    await screen.findByText("Done");
    expect(screen.getByText("2 of 2")).toBeInTheDocument();
    expect(screen.getByText("Ran as approved.")).toBeInTheDocument();
    expect(finishButton()).toBeNull();
  });
});

describe("DeskPlanBody when a step failed", () => {
  function failedPlan() {
    return plan({
      status: "Failed",
      decidedByUserId: "usr_me",
      decidedAt: NOW - 400,
      completedSteps: 1,
      failedStep: 2,
      failureError: "The customer has no email on file.",
    });
  }

  it("says where it stopped, keeps the failed step open with the server's reason", async () => {
    listedPlans = [failedPlan()];
    listedSteps = steps(["Executed", "ExecutionFailed"]);
    renderPlan();

    await screen.findByText("Stopped at step 2");
    expect(screen.getByText("1 of 2 steps went through")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "This step did not go throughThe customer has no email on file.",
    );
    expect(await screen.findByText("Delivery moved")).toBeInTheDocument();
  });

  it("asks the agent to finish it in the conversation, naming the plan, step and reason", async () => {
    listedPlans = [failedPlan()];
    listedSteps = steps(["Executed", "ExecutionFailed"]);
    renderPlan();

    await waitFor(() => expect(finishButton()).toBeEnabled());
    fireEvent.click(finishButton()!);

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      'The plan "Recover S-1001 after the dock delay" stopped at step 2: The customer has no email on file. ' +
        "Step 1 went through. Step 2 did not run. " +
        "Propose the rest of the plan, fixed so it goes through, for me to approve.",
    );
    expect(decideMyPlan).not.toHaveBeenCalled();
    await waitFor(() => expect(finishButton()).toBeDisabled());
  });

  it("does not offer to finish it to someone who does not own the conversation", async () => {
    thread = ownThread({ userId: "usr_someone_else" });
    listedPlans = [failedPlan()];
    listedSteps = steps(["Executed", "ExecutionFailed"]);
    renderPlan();

    await screen.findByText("Stopped at step 2");
    expect(finishButton()).toBeNull();
  });

  it("hands over to a later plan once the agent proposed the rest", async () => {
    listedPlans = [
      failedPlan(),
      plan({ id: "apl_2", title: "Finish recovering S-1001", createdAt: NOW - 60 }),
    ];
    listedSteps = steps(["Executed", "ExecutionFailed"]);
    renderPlan();

    await screen.findByText("Stopped at step 2");
    expect(finishButton()).toBeNull();
    expect(
      screen.getByText("A later plan in this conversation takes it from here."),
    ).toBeInTheDocument();
  });
});

describe("DeskPlanBody when nothing ran", () => {
  it("shows a declined plan with the decider's note as plain words", async () => {
    listedPlans = [
      plan({
        status: "Rejected",
        decidedAt: NOW - 100,
        decidedByUserId: "usr_other",
        decidedByName: "Ana Ruiz",
        decisionNote: "**Not** today",
      }),
    ];
    listedSteps = steps(["Rejected", "Rejected"]);
    renderPlan();

    await screen.findByText("Declined");
    expect(screen.getByText("Ana Ruiz declined the plan")).toBeInTheDocument();
    expect(screen.getByText("**Not** today")).toBeInTheDocument();
    expect(screen.getByText("None of these ran")).toBeInTheDocument();
    expect(finishButton()).toBeNull();
  });

  it("offers to ask again for an expired plan", async () => {
    listedPlans = [plan({ status: "Expired", expiresAt: NOW - 60 })];
    listedSteps = steps(["Expired", "Expired"]);
    renderPlan();

    await screen.findByText("Expired");
    expect(screen.getByText("Nobody decided it")).toBeInTheDocument();
    await waitFor(() => expect(finishButton()).toBeEnabled());
  });

  it("says the plan is gone when the conversation no longer lists it", async () => {
    listedPlans = [];
    listedSteps = [];
    renderPlan();

    expect(
      await screen.findByText("The plan behind this is no longer in the conversation."),
    ).toBeInTheDocument();
  });

  it("draws the plan's title as the plain text the agent gave", async () => {
    listedPlans = [plan({ title: "Recover *S-1001* <b>now</b>" })];
    listedSteps = steps(["Pending", "Pending"]);
    renderPlan();

    expect(await screen.findByText("Recover *S-1001* <b>now</b>")).toBeInTheDocument();
  });
});
