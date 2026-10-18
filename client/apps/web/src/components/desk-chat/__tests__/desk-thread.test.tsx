import type { ThreadAsks, ThreadAskRequest } from "@/components/assistant/ask-requests";
import type { AssistantMessage, AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskThread, type DeskThreadProps } from "../desk-thread";

const conversation = vi.hoisted(() => ({
  reply: "Two loads are stuck at the yard.",
  artifacts: [] as { id: string; kind: string; title: string }[],
  asks: { byReply: new Map(), open: null } as ThreadAsks,
  answer: (() => undefined) as ((value: string) => void) | undefined,
  providersReady: true,
  /** Entries the thread holds after the opening question and reply. */
  more: [] as unknown[],
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    browser: (agentId: string) => ({
      queryKey: ["browser", agentId],
      queryFn: () => ({ available: false }),
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

const question = message({ id: "msg_q", content: "Which loads are stuck?", sequence: 1 });
const reply = message({
  id: "msg_a",
  role: "Assistant",
  content: "Two loads are stuck at the yard.",
  sequence: 2,
});

vi.mock("@/components/assistant/use-thread-model", () => ({
  useThreadModel: () => ({
    entries: [
      { kind: "user", message: question },
      { kind: "assistant", message: { ...reply, content: conversation.reply }, tools: [] },
      ...conversation.more,
    ],
    placements: new Map(),
    turn: null,
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
    current: null,
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
    queue: [],
    resumeAll: () => undefined,
    retry: null,
    send: async () => undefined,
    setProviderId: () => undefined,
    showDock: false,
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

function renderThread(props: Partial<DeskThreadProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/shipments"]}>
        <div className="dk-chat">
          <DeskThread
            thread={thread}
            agent={null}
            agentsUnavailable={false}
            threads={[]}
            {...props}
          />
        </div>
      </MemoryRouter>
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
    const { container } = renderThread({ chapters: { of: () => 0, toggle: () => undefined } });

    const stage = container.querySelector(".dk-stage");
    expect(stage).not.toHaveClass("dk-dense");
    expect(screen.getByText("Which loads are stuck?")).toHaveClass("dk-q");
    expect(screen.getByText("Two loads are stuck at the yard.")).toBeInTheDocument();
    expect(container.querySelector(".dk-time")).not.toBeNull();
    expect(screen.getByRole("button", { name: "Pin as chapter" })).toBeInTheDocument();
    expect(screen.getByText(/Desk can make mistakes/)).toBeInTheDocument();
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

    expect(screen.queryByRole("button", { name: "Pin as chapter" })).toBeNull();
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

    expect(screen.getByRole("textbox", { name: "Message the Desk" })).toHaveAttribute(
      "placeholder",
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
