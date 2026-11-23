import { TestRouter } from "@trenova/shared/testing/test-router";
import { placeholderOf, writeInBox } from "../composer/editor/editor-test-kit";
import type { ThreadAsks, ThreadAskRequest } from "@/components/assistant/ask-requests";
import { approvalQueue } from "@/components/assistant/approval-queue";
import {
  ReplyBoardsProvider,
  type ReplyBoardRenderer,
} from "@/components/assistant/artifact-opener";
import {
  initialTurnState,
  type TurnSegment,
  type TurnState,
} from "@/components/assistant/turn-stream";
import type { AssistantMessage, AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { preview } from "@/components/assistant/__tests__/preview-fixtures";
import { DESK_SETTINGS_DEFAULTS, useDeskSettingsStore } from "@/stores/desk-settings-store";
import { DeskThread, type DeskThreadChapters, type DeskThreadProps } from "../desk-thread";

const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
  decideMyProposals: vi.fn(),
  decideMyPlan: vi.fn(),
  decideAgentProposal: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: async () => preview({ digest: "sha256:shown" }),
}));

const conversation = vi.hoisted(() => ({
  reply: "Two loads are stuck at the yard.",
  artifacts: [] as { id: string; kind: string; title: string }[],
  asks: { byReply: new Map(), open: null } as ThreadAsks,
  answer: (() => undefined) as ((value: string) => void) | undefined,
  providersReady: true,
  /** Entries the thread holds after the opening question and reply. */
  more: [] as unknown[],
  /** The decision the dock asks, and the whole queue it was picked from. */
  queue: [] as unknown[],
  /** The turn being written, or null when none runs. */
  turn: null as unknown,
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    mentionSearch: (query: string, tab: string) => ({
      queryKey: ["mention-search", query, tab],
      queryFn: () => [],
    }),
    commands: (agentId: string) => ({
      queryKey: ["assistant-commands", agentId],
      queryFn: () => ({ commands: [] }),
    }),
    agentMatch: (text: string) => ({
      queryKey: ["agent-match", text],
      queryFn: () => ({ outcome: "NoMatch", agent: null, tools: [] }),
    }),
    artifacts: (threadId: string) => ({
      queryKey: ["artifacts", threadId],
      queryFn: () => ({ results: [] }),
      _ctx: {
        summary: {
          queryKey: ["artifacts", threadId, "summary"],
          queryFn: () => ({ results: conversation.artifacts }),
        },
      },
    }),
    threadBudget: (threadId: string) => ({
      queryKey: ["budget", threadId],
      queryFn: () => new Promise(() => undefined),
    }),
    myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
    providers: () => ({ queryKey: ["providers"], queryFn: () => [] }),
  },
}));

vi.mock("../composer/desk-capture", () => ({
  useDeskScans: () => null,
  DeskCapturePanel: () => null,
}));

vi.mock("@/components/assistant/use-askable-agent", () => ({
  useAskableAgent: () => ({
    agent: null,
    choose: () => undefined,
    recency: { ids: [], lastUsedAt: new Map() },
    choices: { isLoading: false, isError: false, recent: [], items: [], refetch: () => undefined },
    noneAvailable: false,
  }),
}));

vi.mock("@/components/assistant/use-composer-context", () => ({
  useComposerContext: () => ({
    attachments: [],
    attachFiles: () => undefined,
    removeAttachment: () => undefined,
    retryAttachment: () => undefined,
    mentions: [],
    setMentions: () => undefined,
    searchMentions: async () => [],
    clear: () => undefined,
  }),
}));

function message(overrides: Partial<AssistantMessage>): AssistantMessage {
  return {
    id: "msg",
    threadId: "athr_1",
    role: "User",
    kind: "Message",
    content: "",
    sequence: 1,
    createdAt: Math.floor(Date.now() / 1000) - 60,
    ...overrides,
  } as AssistantMessage;
}

const question = message({
  id: "msg_q",
  content: "Which loads are stuck?",
  sequence: 1,
  turnId: "atrn_1",
});
const reply = message({
  id: "msg_a",
  role: "Assistant",
  content: "Two loads are stuck at the yard.",
  sequence: 2,
  turnId: "atrn_1",
});

vi.mock("@/components/assistant/use-thread-model", () => ({
  useThreadModel: () => ({
    entries: [
      { kind: "user", message: question },
      { kind: "assistant", message: { ...reply, content: conversation.reply }, tools: [] },
      ...conversation.more,
    ],
    placements: new Map(),
    turn: conversation.turn,
    isActive: false,
    compaction: {
      usage: null,
      auto: true,
      setAuto: async () => undefined,
      compact: async () => undefined,
      compacting: null,
      cancel: () => undefined,
    },
    older: { has: false, loading: false, load: () => undefined },
    answer: conversation.answer,
    asks: conversation.asks,
    block: null,
    current: conversation.queue.length > 0 ? { entry: conversation.queue[0], index: 0 } : null,
    decided: () => undefined,
    deferAll: () => undefined,
    dismiss: async () => undefined,
    draft: "",
    followUpDecision: () => undefined,
    onDraftChange: () => undefined,
    proposalsQuery: { data: { results: [] } },
    providerId: "",
    providers: [],
    providersReady: conversation.providersReady,
    queue: conversation.queue,
    resumeAll: () => undefined,
    retry: null,
    send: async () => undefined,
    setProviderId: () => undefined,
    showDock: conversation.queue.length > 0,
    stop: () => undefined,
    suggestions: [],
    waits: { open: [], byId: new Map(), cancel: async () => undefined },
    waiting: {
      items: [],
      add: async () => true,
      edit: async () => true,
      remove: async () => undefined,
      move: async () => undefined,
      sendNow: async () => undefined,
    },
  }),
}));

afterEach(() => {
  cleanup();
  conversation.reply = "Two loads are stuck at the yard.";
  conversation.artifacts = [];
  conversation.asks = { byReply: new Map(), open: null };
  conversation.answer = () => undefined;
  conversation.providersReady = true;
  conversation.more = [];
  conversation.queue = [];
  conversation.turn = null;
});

const thread = {
  id: "athr_1",
  title: "Stuck loads",
  agentDefinitionId: "agdef_1",
  canContinue: true,
  status: "Active",
  lastMessageAt: 0,
  createdAt: 0,
} as AssistantThread;

/** Chapters as the Desk passes them: a heading for each turn named, and a rail that says which is in view. */
function chaptersOf(
  titles: Record<string, string>,
  start: (messageId: string) => void = () => undefined,
): DeskThreadChapters {
  return {
    heading: (turnId) => (titles[turnId] ? <p>{titles[turnId]}</p> : null),
    liveTurnId: null,
    rail: (nav) => <span>{`rail at ${nav.current ?? "none"}`}</span>,
    startAt: (entry) => (entry.turnId && titles[entry.turnId] ? undefined : () => start(entry.id)),
  };
}

function renderThread(props: Partial<DeskThreadProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter initialEntries={["/shipments"]}>
        <div className="dk-chat">
          <DeskThread
            thread={thread}
            agent={null}
            agentsUnavailable={false}
            threads={[]}
            {...props}
          />
        </div>
      </TestRouter>
    </QueryClientProvider>,
  );
}

/**
 * The Desk and the assistant draw a conversation with the same thread. The
 * Desk's room sets it in three columns, with each reply's time in the gutter;
 * the assistant's panel sets the same turns in one compact column. What only
 * the Desk has is there only when the Desk passes it.
 */
describe("DeskThread in the Desk's density", () => {
  it("draws the question as a heading and the reply under it, with its time in the gutter", () => {
    const { container } = renderThread({ chapters: chaptersOf({}) });

    const stage = container.querySelector(".dk-stage");
    expect(stage).not.toHaveClass("dk-dense");
    expect(screen.getByText("Which loads are stuck?")).toHaveClass("dk-q");
    expect(screen.getByText("Two loads are stuck at the yard.")).toBeInTheDocument();
    expect(container.querySelector(".dk-time")).not.toBeNull();
    expect(screen.getByRole("button", { name: "Start a chapter here" })).toBeInTheDocument();
    expect(screen.getByText(/Desk can make mistakes/)).toBeInTheDocument();
  });

  it("sets a chapter's heading above the question that opens it and the rail in the corner", () => {
    const started = vi.fn();
    const { container } = renderThread({
      chapters: chaptersOf({ atrn_1: "Stuck loads at the yard" }, started),
    });

    const heading = container.querySelector('[data-chapter-turn="atrn_1"]');
    // The row is display: contents; the turn sits on its cell, which has a box to scroll to.
    expect(heading).toHaveClass("dk-c");
    expect(heading).toHaveTextContent("Stuck loads at the yard");
    expect(heading?.parentElement?.nextElementSibling).toHaveTextContent("Which loads are stuck?");
    expect(container.querySelector(".dk-chap-rail-slot")).toHaveTextContent("rail at atrn_1");
    expect(screen.queryByRole("button", { name: "Start a chapter here" })).toBeNull();
    expect(started).not.toHaveBeenCalled();
  });

  it("starts a chapter where the reply was asked", () => {
    const started = vi.fn();
    renderThread({ chapters: chaptersOf({}, started) });

    fireEvent.click(screen.getByRole("button", { name: "Start a chapter here" }));

    expect(started).toHaveBeenCalledWith("msg_a");
  });

  it("keeps a pane beside the thread for the workspace it is given", () => {
    const { container } = renderThread({
      artifacts: {
        open: () => undefined,
        workspace: { open: true, setOpen: () => undefined, content: <p>Workspace</p> },
      },
    });

    expect(container.querySelector(".dk-stage")).toHaveClass("dk-open");
    expect(screen.getByRole("complementary", { name: "Artifacts" })).toHaveTextContent("Workspace");
  });
});

describe("DeskThread in the compact density", () => {
  it("sets the same turns in one dense column", () => {
    const { container } = renderThread({
      density: "compact",
      disclaimer: "The assistant can make mistakes.",
    });

    expect(container.querySelector(".dk-stage")).toHaveClass("dk-dense");
    expect(screen.getByText("Which loads are stuck?")).toHaveClass("dk-q");
    expect(screen.getByText("Two loads are stuck at the yard.")).toBeInTheDocument();
    expect(screen.getByText("The assistant can make mistakes.")).toBeInTheDocument();
  });

  it("leaves out what only the Desk has", () => {
    const { container } = renderThread({ density: "compact" });

    expect(screen.queryByRole("button", { name: "Start a chapter here" })).toBeNull();
    expect(screen.getByRole("button", { name: "Copy reply" })).toBeInTheDocument();
    expect(screen.queryByRole("complementary", { name: "Artifacts" })).toBeNull();
    expect(container.querySelector(".dk-stage")).not.toHaveClass("dk-open");
  });

  it("shows the page chip for the page on screen, and none for a thread bound to its page", () => {
    renderThread({ density: "compact", pageSource: "screen" });
    expect(screen.getByRole("button", { name: "Sharing /shipments" })).toBeInTheDocument();
    cleanup();

    renderThread({ density: "compact", pageSource: "none" });
    expect(screen.queryByRole("button", { name: /^(Sharing|Not sharing)/ })).toBeNull();
  });
});

describe("DeskThread's artifact links", () => {
  it("opens the artifact a saved reply names from its badge", async () => {
    conversation.reply =
      "The [Shipments](artifact:art_01M4PB27N0HX4MVKQJN5NXPTV7) shows 15 completed shipments.";
    conversation.artifacts = [
      { id: "art_01M4PB27N0HX4MVKQJN5NXPTV7", kind: "table_view", title: "Shipments" },
    ];
    const open = vi.fn();
    renderThread({ artifacts: { open } });

    const badge = await screen.findByRole("button", { name: "Open Shipments" });
    expect(badge.closest("p")).toHaveTextContent("The Shipments shows 15 completed shipments.");
    fireEvent.click(badge);
    expect(open).toHaveBeenCalledWith("art_01M4PB27N0HX4MVKQJN5NXPTV7");
  });
});

describe("DeskThread's open question", () => {
  const request: ThreadAskRequest = {
    sequence: 3,
    callId: "call_3",
    question: "Which window should the report cover?",
    options: [
      { value: "7", label: "7 days", detail: "the last week" },
      { value: "30", label: "30 days", detail: "" },
    ],
    allowOther: true,
    otherHint: "Number of days",
    suggestion: "",
    input: { type: "text" },
    fields: [],
  };

  /** The reply asked the question and nothing has answered it. */
  function asked() {
    conversation.asks = {
      byReply: new Map([["msg_a", [{ request, answer: null }]]]),
      open: request,
    };
    // A conversation with no model to answer is locked, and a locked one
    // docks nothing; the models are left unread here so it is not.
    conversation.providersReady = false;
  }

  it("docks the question above the composer and leaves a line where it was asked", () => {
    asked();
    const { container } = renderThread();

    const dock = container.querySelector(".dk-dock");
    const question = screen.getByRole("region", { name: "Question from the agent" });
    expect(dock).toContainElement(question);
    expect(question).toHaveTextContent("Which window should the report cover?");
    expect(screen.getByText("Waiting for your answer below")).toBeInTheDocument();
  });

  it("makes the composer the box for a value in words", () => {
    asked();
    renderThread();

    expect(placeholderOf(screen.getByRole("textbox", { name: "Message the Desk" }))).toBe(
      "Number of days",
    );
  });

  it("offers the agent's suggested answer in the composer", () => {
    const partnerCode: ThreadAskRequest = {
      ...request,
      question: "What partner code should Meritor have in Trenova?",
      options: [],
      otherHint: "A short code, such as MERITOR",
      suggestion: "MERITOR",
    };
    conversation.asks = {
      byReply: new Map([["msg_a", [{ request: partnerCode, answer: null }]]]),
      open: partnerCode,
    };
    conversation.providersReady = false;
    renderThread();

    const box = screen.getByRole("textbox", { name: "Message the Desk" });
    expect(box).toHaveAccessibleDescription("Suggested answer: MERITOR. Press Tab to use it.");
    expect(
      screen.getByText("Press Tab to use the suggested answer, or type your own below."),
    ).toBeInTheDocument();
  });

  it("answers with the choice picked", () => {
    asked();
    const answer = vi.fn();
    conversation.answer = answer;
    renderThread();

    fireEvent.click(screen.getByRole("button", { name: "30 days" }));

    expect(answer).toHaveBeenCalledWith("30");
  });

  // The record of the question shows what the person answered; the message
  // that carried it is not drawn again as a new exchange.
  it("reads an answer under its question rather than as a new heading", () => {
    const answer = message({ id: "msg_ans", content: "Host: sftp.meritor.example", sequence: 4 });
    conversation.more = [{ kind: "user", message: answer }];
    conversation.asks = {
      byReply: new Map([["msg_a", [{ request, answer }]]]),
      open: null,
    };
    const { container } = renderThread();

    expect(screen.getByText("Host: sftp.meritor.example").closest(".dk-ask-rec")).not.toBeNull();
    expect(container.querySelectorAll(".dk-q")).toHaveLength(1);
  });

  it("keeps the files an answer carried, under its question", () => {
    const answer = message({
      id: "msg_ans",
      content: "Here is the guide",
      sequence: 4,
      attachments: [{ documentId: "doc_1", fileName: "Meritor-EDI-204.pdf", fileSize: 2048 }],
    });
    conversation.more = [{ kind: "user", message: answer }];
    conversation.asks = {
      byReply: new Map([["msg_a", [{ request, answer }]]]),
      open: null,
    };
    const { container } = renderThread();

    expect(container.querySelectorAll(".dk-q")).toHaveLength(1);
    expect(screen.getByText("Meritor-EDI-204.pdf").closest(".dk-ma")).not.toBeNull();
  });

  it("docks nothing in a conversation nobody can send to", () => {
    asked();
    conversation.answer = undefined;
    renderThread();

    expect(screen.queryByRole("region", { name: "Question from the agent" })).toBeNull();
    expect(screen.getByText("Not answered yet")).toBeInTheDocument();
  });
});

/*
A turn that raised a change while the organization's agents are paused
said "it waits on your approval" and docked nothing: the change was held,
and a held change was dropped from the queue. The dock now asks it, naming
the pause in place of the approval.
*/
describe("DeskThread's dock for a held change", () => {
  it("asks the held change in the composer's place, with nothing to approve", () => {
    conversation.queue = approvalQueue(
      [
        {
          id: "ap_1",
          runId: "ar_1",
          toolName: "home_add_widget",
          arguments: { key: "fleet-status", version: 0 },
          rationale: "",
          autonomyTier: "ActWithApproval",
          status: "Pending",
          sourceMessageId: "msg_a",
          executedAt: null,
          executionError: "",
          expiresAt: 0,
          hold: { reason: "OrganizationPaused", agentName: "" },
          planId: "",
          planStep: 0,
          fields: [],
          createdAt: 100,
          decidedAt: null,
          decidedByUserId: "",
          decisionNote: "",
          reviewRequired: false,
        },
      ],
      [],
      null,
      1_000,
    );

    renderThread();

    const card = screen.getByRole("group", { name: "Held, not yet decidable" });
    expect(card.textContent).toContain("Agents are paused for the organization");
    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    expect(screen.queryByText(/to approve/)).toBeNull();
  });
});

/*
A schedule the person approved from an agent's proposal is kept as a card
worded as the request the agent wrote out. Drawn like one they typed, it read
as their own words. It is drawn as its card alone; one they typed keeps their
words above it.
*/
describe("DeskThread's schedule cards", () => {
  function schedules() {
    return {
      isRequest: () => false,
      create: () => undefined,
      pending: null,
      card: (scheduleId: string, proposed: boolean) => (
        <div>{`card ${scheduleId} ${proposed ? "proposed" : "typed"}`}</div>
      ),
    };
  }

  it("draws a schedule approved from a proposal as its card, not as the person's words", () => {
    conversation.more = [
      {
        kind: "schedule",
        message: message({
          id: "msg_s",
          kind: "Schedule",
          content: "Every day at 10:15 AM, Give me today's operations board",
          scheduleId: "csch_1",
          proposalId: "ap_1",
          sequence: 3,
        }),
      },
    ];
    renderThread({ schedules: schedules() });

    expect(screen.getByText("card csch_1 proposed")).toBeInTheDocument();
    expect(
      screen.queryByText("Every day at 10:15 AM, Give me today's operations board"),
    ).toBeNull();
  });

  it("keeps the words of a schedule the person typed above its card", () => {
    conversation.more = [
      {
        kind: "schedule",
        message: message({
          id: "msg_s",
          kind: "Schedule",
          content: "every day at 10:15am, give me today's operations board",
          scheduleId: "csch_2",
          sequence: 3,
        }),
      },
    ];
    renderThread({ schedules: schedules() });

    expect(screen.getByText("card csch_2 typed")).toBeInTheDocument();
    expect(screen.getByText("every day at 10:15am, give me today's operations board")).toHaveClass(
      "dk-q",
    );
  });
});

/*
With "Send with Cmd Enter" chosen, the message box sends on Cmd/Ctrl+Enter, and the
stage approved on the same key without asking whether the box had used it: a
person typing "why this driver?" about a waiting change sent the question and
approved the change in one keystroke.
*/
describe("DeskThread approves only on purpose", () => {
  function waitingChange() {
    conversation.queue = approvalQueue(
      [
        {
          id: "ap_1",
          runId: "ar_1",
          toolName: "shipment_update",
          arguments: { shipmentId: "shp_1", status: "Delayed" },
          rationale: "",
          autonomyTier: "ActWithApproval",
          status: "Pending",
          sourceMessageId: "msg_a",
          executedAt: null,
          executionError: "",
          expiresAt: 0,
          hold: null,
          planId: "",
          planStep: 0,
          fields: [],
          createdAt: 100,
          decidedAt: null,
          decidedByUserId: "",
          decisionNote: "",
          reviewRequired: false,
        },
      ],
      [],
      null,
      1_000,
    );
  }

  afterEach(() => {
    decideMyProposal.mockReset();
    useDeskSettingsStore.setState({ settings: DESK_SETTINGS_DEFAULTS });
  });

  it("sends a question with Ctrl+Enter without approving the change it asks about", async () => {
    useDeskSettingsStore.setState({ settings: { ...DESK_SETTINGS_DEFAULTS, send: "mod" } });
    waitingChange();
    // No lock for a missing model: these are about the box that is there.
    conversation.providersReady = false;
    renderThread();
    await waitFor(() => expect(screen.getByRole("button", { name: /^Approve/ })).toBeEnabled());

    const box = screen.getByRole("textbox", { name: "Message the Desk" });
    writeInBox(box, "Why this driver?");
    fireEvent.keyDown(box, { key: "Enter", ctrlKey: true });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(decideMyProposal).not.toHaveBeenCalled();
  });

  it("approves with Ctrl+Enter from an empty message box", async () => {
    decideMyProposal.mockResolvedValue({
      proposalId: "ap_1",
      executed: false,
      refused: false,
      error: null,
      decision: { commitsAt: null },
    });
    waitingChange();
    // No lock for a missing model: these are about the box that is there.
    conversation.providersReady = false;
    renderThread();
    await waitFor(() => expect(screen.getByRole("button", { name: /^Approve/ })).toBeEnabled());

    fireEvent.keyDown(screen.getByRole("textbox", { name: "Message the Desk" }), {
      key: "Enter",
      ctrlKey: true,
    });

    await waitFor(() => expect(decideMyProposal).toHaveBeenCalledTimes(1));
  });
});

/** How the Desk draws boards, as plain stand-ins: the saved boards by id, the running turn's skeleton. */
const STAND_IN_BOARDS: ReplyBoardRenderer = {
  Boards: ({ boards }) => <p>Board {boards.map((item) => item.artifact.id).join(", ")}</p>,
  Pending: ({ tool }) => (
    <div aria-busy="true" aria-label="Reading the board…">
      Skeleton of {tool}
    </div>
  ),
};

function renderDeskThread() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const tree = () => (
    <QueryClientProvider client={queryClient}>
      <TestRouter initialEntries={["/shipments"]}>
        <ReplyBoardsProvider renderer={STAND_IN_BOARDS}>
          <div className="dk-chat">
            <DeskThread thread={thread} agent={null} agentsUnavailable={false} threads={[]} />
          </div>
        </ReplyBoardsProvider>
      </TestRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return { ...view, redraw: () => view.rerender(tree()) };
}

const NOTE = "I'll find the open GlobalTrade loads.";
const SECOND_NOTE = "Found four of them.";
const ANSWER_ARGS = {
  title: "Open GlobalTrade loads",
  answer: "There are {count} open loads.",
  groups: [{ title: "Open", records: ["shp_1"] }],
};

/** A turn being written, as the stream builds it (`turn-stream.ts`). */
function runningTurn(segments: TurnSegment[]): TurnState {
  return {
    ...initialTurnState("Which GlobalTrade loads are open?"),
    status: "working",
    segments,
  };
}

const lookup: TurnSegment = {
  kind: "tool",
  callId: "call_lookup",
  name: "search_records",
  arguments: {},
  status: "done",
  content: "",
};

const answering: TurnSegment = {
  kind: "tool",
  callId: "call_art_a",
  name: "show_answer",
  arguments: ANSWER_ARGS,
  status: "running",
  content: "",
};

/** The same turn once saved: the notes, the step that drew the board, the reply that links it. */
function savedBoardTurn() {
  conversation.reply = NOTE;
  conversation.more = [
    {
      kind: "assistant",
      message: message({ id: "msg_s2", role: "Assistant", content: SECOND_NOTE, sequence: 3 }),
      tools: [
        { call: { id: "call_art_a", name: "show_answer", arguments: ANSWER_ARGS }, result: null },
      ],
    },
    {
      kind: "assistant",
      message: message({
        id: "msg_final",
        role: "Assistant",
        content: "There are 4 open loads ([Open GlobalTrade loads](artifact:art_a)).",
        sequence: 4,
      }),
      tools: [],
    },
  ];
}

/*
A board placed in the conversation is the reply; the notes the agent wrote on
the way to it are not. While the turn runs they stream as words, since nothing
says yet which step is the answer. The moment the board is known they fold away
under its skeleton (out of reach while they go, then gone), so when the turn is
saved, and the saved turn leaves them out, nothing on the page moves. jsdom
draws no layout, so the fold's motion itself (height and opacity closing over
220ms) is not asserted here; that it starts, is out of reach and ends is.
*/
/** The running turn's words as drawn, each streamed run of prose. */
function streamedProse(): Element[] {
  return [...document.querySelectorAll(".dk-prose.dk-streaming")];
}

describe("the notes before a board in the conversation", () => {
  it("streams the notes as the reply while no board is known", () => {
    conversation.turn = runningTurn([{ kind: "text", text: `${NOTE} `, closed: true }, lookup]);
    renderDeskThread();

    const [note] = streamedProse();
    expect(streamedProse().map((prose) => prose.textContent)).toEqual([NOTE]);
    expect(note.closest("[inert]")).toBeNull();
    expect(note.closest("[aria-hidden]")).toBeNull();
    expect(screen.queryByLabelText("Reading the board…")).not.toBeInTheDocument();
  });

  it("folds the notes away under the board's skeleton the moment the board is known, out of reach while they go", async () => {
    conversation.turn = runningTurn([{ kind: "text", text: `${NOTE} `, closed: true }, lookup]);
    const { redraw } = renderDeskThread();
    const [streaming] = streamedProse();

    conversation.turn = runningTurn([
      { kind: "text", text: `${NOTE} `, closed: true },
      lookup,
      { kind: "text", text: SECOND_NOTE, closed: true },
      answering,
    ]);
    redraw();

    const [leaving] = streamedProse();
    expect(streamedProse()).toHaveLength(1);
    expect(leaving).toBe(streaming);
    expect(leaving.textContent).toBe(`${NOTE} ${SECOND_NOTE}`);
    expect(leaving.closest("[inert]")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByLabelText("Reading the board…")).toHaveTextContent(
      "Skeleton of show_answer",
    );

    await waitFor(() => expect(streamedProse()).toHaveLength(0));
    expect(document.querySelector("[inert]")).toBeNull();
    expect(screen.getByLabelText("Reading the board…")).toBeInTheDocument();
  });

  it("draws no notes when a turn already placing its board is first drawn", () => {
    conversation.turn = runningTurn([
      { kind: "text", text: `${NOTE} `, closed: true },
      lookup,
      answering,
    ]);
    renderDeskThread();

    expect(streamedProse()).toHaveLength(0);
    expect(document.querySelector("[inert]")).toBeNull();
    expect(screen.getByLabelText("Reading the board…")).toBeInTheDocument();
  });

  it("draws the saved turn as its board alone, with none of the notes", async () => {
    savedBoardTurn();
    conversation.artifacts = [
      {
        id: "art_a",
        kind: "board",
        title: "Open GlobalTrade loads",
        payload: { place: "conversation" },
        sourceToolCallId: "call_art_a",
        lineageId: "",
        lineageSeq: 1,
      } as unknown as { id: string; kind: string; title: string },
    ];
    renderDeskThread();

    expect(await screen.findByText("Board art_a")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(NOTE);
    expect(document.body).not.toHaveTextContent(SECOND_NOTE);
    expect(document.querySelector("[inert]")).toBeNull();
  });

  it("hands the running turn over to the saved one with the notes never coming back", () => {
    conversation.turn = runningTurn([
      { kind: "text", text: `${NOTE} `, closed: true },
      lookup,
      answering,
    ]);
    const { redraw } = renderDeskThread();
    expect(document.body).not.toHaveTextContent(NOTE);

    savedBoardTurn();
    redraw();

    expect(screen.getByText("Board art_a")).toBeInTheDocument();
    expect(screen.queryByLabelText("Reading the board…")).not.toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(NOTE);
    expect(document.body).not.toHaveTextContent(SECOND_NOTE);
  });
});
