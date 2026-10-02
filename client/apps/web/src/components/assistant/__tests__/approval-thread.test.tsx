import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import { useAssistantStore } from "@/stores/assistant-store";
import {
  assistantMessageSchema,
  assistantPlanListSchema,
  assistantProposalListSchema,
  assistantThreadSchema,
  type AssistantMessage,
} from "@/types/assistant";
import { MessageThread } from "../message-thread";
import { preview } from "./preview-fixtures";

const api = vi.hoisted(() => ({
  listProposals: vi.fn(),
  listPlans: vi.fn(),
  listProviders: vi.fn(),
}));
const turn = vi.hoisted(() => ({ isActive: false, send: vi.fn(), rejoin: vi.fn() }));
const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      listProposals: api.listProposals,
      listPlans: api.listPlans,
      listProviders: api.listProviders,
    },
  },
}));

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
}));

const history = vi.hoisted(() => ({ messages: [] as unknown[] }));

vi.mock("../use-thread-history", () => ({
  useThreadHistory: () => ({
    messages: history.messages,
    isLoading: false,
    hasOlder: false,
    isLoadingOlder: false,
    loadOlder: vi.fn(),
    total: history.messages.length,
    limit: 0,
    length: { state: "open", remaining: Number.POSITIVE_INFINITY },
  }),
}));

vi.mock("../use-assistant-turn", () => ({
  useAssistantTurn: () => ({
    turn: null,
    isActive: turn.isActive,
    send: turn.send,
    rejoin: turn.rejoin,
    stop: vi.fn(),
    dismiss: vi.fn(),
    retry: undefined,
  }),
}));

vi.mock("@/components/ai-feedback/feedback-control", () => ({ FeedbackControl: () => null }));

vi.mock("../use-active-turns", () => ({ useLiveThreadIds: () => new Set<string>() }));

vi.mock("../use-page-context", () => ({ usePageContext: () => () => null }));

vi.mock("../use-composer-context", () => ({
  useComposerContext: () => ({
    attachments: [],
    attachFiles: vi.fn(),
    removeAttachment: vi.fn(),
    mentions: [],
    setMentions: vi.fn(),
    searchMentions: vi.fn(async () => []),
    clear: vi.fn(),
  }),
}));

// The window lays out nothing without a size; every row is drawn here.
vi.mock("../virtual-thread", () => ({
  VirtualThread: ({ rows }: { rows: { key: string; render: () => React.ReactNode }[] }) => (
    <div>
      {rows.map((row) => (
        <div key={row.key}>{row.render()}</div>
      ))}
    </div>
  ),
}));

const ME = "usr_me";

/* services.AssistantThread, services.AssistantProposal and the message as the thread routes serve them. */
const thread = assistantThreadSchema.parse({
  id: "athr_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: ME,
  agentDefinitionId: "agdef_1",
  title: "Billing",
  status: "Active",
  canContinue: true,
  createdAt: 10,
  updatedAt: 10,
});

const agent = {
  id: "agdef_1",
  name: "Billing assistant",
  description: "",
  starters: [],
} as unknown as AgentChoice;

function message(overrides: Record<string, unknown>): AssistantMessage {
  return assistantMessageSchema.parse({
    threadId: "athr_1",
    createdAt: 1_790_000_000,
    ...overrides,
  });
}

function serverProposal(overrides: Record<string, unknown> = {}) {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "post_invoice",
    arguments: { invoiceId: "inv_1" },
    rationale: "",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_2",
    confidence: 0.9,
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    hold: null,
    planId: "",
    planStep: 0,
    simulatedAt: null,
    simulation: null,
    fields: [],
    modifications: null,
    createdAt: 1_790_000_000,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    ...overrides,
  };
}

function serve(proposals: Record<string, unknown>[]) {
  api.listProposals.mockResolvedValue(assistantProposalListSchema.parse({ results: proposals }));
}

beforeEach(() => {
  history.messages = [
    message({ id: "amsg_1", sequence: 1, role: "User", content: "Post the March invoices" }),
    message({ id: "amsg_2", sequence: 2, role: "Assistant", content: "I proposed it." }),
  ];
  api.listPlans.mockResolvedValue(assistantPlanListSchema.parse({ results: [] }));
  api.listProviders.mockResolvedValue([]);
  fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
  decideMyProposal.mockResolvedValue({});
  turn.isActive = false;
  useAssistantStore.setState({ deferredDecisions: [], decisionFocus: {}, drafts: {} });
  useAuthStore.setState({ user: { id: ME, timezone: "UTC" } as never });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderThread(expanded: boolean) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MessageThread thread={thread} agent={agent} expanded={expanded} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const composerBox = () => screen.queryByPlaceholderText("Ask Billing assistant…");

describe.each([
  ["the Desk", true],
  ["the floating panel", false],
])("a conversation with a decision waiting, on %s", (_surface, expanded) => {
  it("asks it in the composer's place and keeps only a line for it in the transcript", async () => {
    serve([serverProposal()]);
    renderThread(expanded);

    expect(await screen.findByRole("region", { name: "Post invoice" })).toBeInTheDocument();
    expect(composerBox()).toBeNull();
    expect(screen.getByText(/Proposed: /)).toBeInTheDocument();
    expect(screen.getByText(/Waiting — decide below/)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: /^approve$/i })).toHaveLength(1);
  });

  it("collapses to a pill above the composer, keeps the draft, and reopens", async () => {
    const user = userEvent.setup();
    serve([serverProposal()]);
    useAssistantStore.setState({ drafts: { athr_1: "Also check April" } });
    renderThread(expanded);

    await user.click(await screen.findByRole("button", { name: "Decide later" }));

    // The box leaves on its own motion; the composer stands once it has gone.
    await waitFor(() => expect(screen.queryByRole("region", { name: "Post invoice" })).toBeNull());
    expect(composerBox()).toHaveValue("Also check April");
    expect(useAssistantStore.getState().deferredDecisions).toEqual(["proposal:aprop_1"]);

    await user.click(screen.getByRole("button", { name: /1 decision waiting/ }));

    expect(await screen.findByRole("region", { name: "Post invoice" })).toBeInTheDocument();
    expect(composerBox()).toBeNull();
    expect(useAssistantStore.getState().drafts.athr_1).toBe("Also check April");
  });

  it("offers the way back to the approval box from the record in the transcript", async () => {
    const user = userEvent.setup();
    serve([serverProposal()]);
    renderThread(expanded);

    await user.click(await screen.findByRole("button", { name: "Decide later" }));
    await waitFor(() => expect(screen.queryByRole("region", { name: "Post invoice" })).toBeNull());

    await user.click(screen.getByRole("button", { name: "Open in approval box" }));

    expect(await screen.findByRole("region", { name: "Post invoice" })).toBeInTheDocument();
    expect(useAssistantStore.getState().deferredDecisions).toEqual([]);
  });
});

describe("the approval box in a conversation", () => {
  it("asks the oldest first, says how many wait, and moves on once it is decided", async () => {
    const user = userEvent.setup();
    serve([
      serverProposal({
        id: "aprop_late",
        runId: "arun_2",
        toolName: "send_invoice",
        createdAt: 1_790_000_200,
      }),
      serverProposal({ id: "aprop_early", runId: "arun_1", createdAt: 1_790_000_100 }),
    ]);
    renderThread(true);

    expect(await screen.findByRole("region", { name: "Post invoice" })).toHaveTextContent("1 of 2");

    serve([
      serverProposal({
        id: "aprop_late",
        runId: "arun_2",
        toolName: "send_invoice",
        createdAt: 1_790_000_200,
      }),
      serverProposal({
        id: "aprop_early",
        runId: "arun_1",
        status: "Executed",
        executedAt: 1_790_000_300,
        decidedAt: 1_790_000_290,
        decidedByUserId: ME,
        createdAt: 1_790_000_100,
      }),
    ]);
    await waitFor(() => expect(screen.getByRole("button", { name: /^approve$/i })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: /^approve$/i }));

    expect(await screen.findByRole("region", { name: "Send invoice" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Send invoice" })).not.toHaveTextContent("of 2");
    expect(decideMyProposal).toHaveBeenCalledWith(
      "aprop_early",
      expect.objectContaining({ decision: "Accepted", previewDigest: "sha256:shown" }),
    );
    expect(turn.rejoin).toHaveBeenCalled();
    expect(turn.send).not.toHaveBeenCalled();
  });

  // Telling the agent is the rejection's own follow-up: nothing is sent as
  // a message, so the agent answers once.
  it("tells the agent through the rejection, never as a second message", async () => {
    const user = userEvent.setup();
    serve([serverProposal()]);
    renderThread(true);

    await user.click(await screen.findByRole("button", { name: "Tell the agent instead" }));
    await user.type(screen.getByLabelText(/What should the agent do instead/), "Wait for April");
    await user.click(screen.getByRole("button", { name: "Send to the agent" }));

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Rejected", note: "Wait for April" }),
      ),
    );
    expect(turn.send).not.toHaveBeenCalled();
    await waitFor(() => expect(turn.rejoin).toHaveBeenCalled());
  });

  it("keeps the composer while the agent is writing", async () => {
    turn.isActive = true;
    serve([serverProposal()]);
    renderThread(true);

    expect(await screen.findByText(/Waiting — decide below/)).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Post invoice" })).toBeNull();
    expect(composerBox()).not.toBeNull();
  });

  it("opens on the decision a request focused, even one put off", async () => {
    serve([
      serverProposal({ id: "aprop_early", runId: "arun_1", createdAt: 1_790_000_100 }),
      serverProposal({
        id: "aprop_late",
        runId: "arun_2",
        toolName: "send_invoice",
        createdAt: 1_790_000_200,
      }),
    ]);
    useAssistantStore.setState({
      deferredDecisions: ["proposal:aprop_early", "proposal:aprop_late"],
    });
    renderThread(true);

    expect(await screen.findByRole("button", { name: /2 decisions waiting/ })).toBeInTheDocument();

    act(() => {
      useAssistantStore
        .getState()
        .focusDecision("athr_1", { proposalIds: ["aprop_late"], planId: "" }, [
          "proposal:aprop_late",
        ]);
    });

    expect(await screen.findByRole("region", { name: "Send invoice" })).toHaveTextContent("2 of 2");
  });
});

describe("the transcript's record of a decision", () => {
  it("says who decided it and opens onto what it did, with nothing to decide", async () => {
    const user = userEvent.setup();
    serve([
      serverProposal({
        status: "Rejected",
        decidedAt: 1_790_000_100,
        decidedByUserId: ME,
        decisionNote: "Wait for April",
      }),
    ]);
    renderThread(true);

    const line = await screen.findByRole("button", { name: /Proposed: .*Rejected by you/ });
    expect(screen.queryByRole("region")).toBeNull();
    expect(composerBox()).not.toBeNull();

    await user.click(line);

    expect(await screen.findByText("You told the agent: “Wait for April”")).toBeInTheDocument();
    expect(screen.getByText("Rejected. Nothing was changed.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^approve$/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /^reject$/i })).toBeNull();
  });

  it("says when someone else decided it", async () => {
    serve([
      serverProposal({
        status: "Executed",
        executedAt: 1_790_000_200,
        decidedAt: 1_790_000_100,
        decidedByUserId: "usr_someone_else",
      }),
    ]);
    renderThread(true);

    const line = await screen.findByRole("button", { name: /Proposed: / });
    expect(line).toHaveTextContent(/Approved/);
    expect(line).not.toHaveTextContent(/by you/);
  });
});
