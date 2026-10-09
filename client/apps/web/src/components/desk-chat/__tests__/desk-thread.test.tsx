import type { AssistantMessage, AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskThread, type DeskThreadProps } from "../desk-thread";

vi.mock("@/lib/queries", () => ({
  queries: {
    assistant: {
      artifacts: (threadId: string) => ({
        queryKey: ["artifacts", threadId],
        queryFn: () => ({ results: [] }),
      }),
      threadBudget: (threadId: string) => ({
        queryKey: ["budget", threadId],
        queryFn: () => new Promise(() => undefined),
      }),
      myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
      providers: () => ({ queryKey: ["providers"], queryFn: () => [] }),
    },
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
      { kind: "assistant", message: reply, tools: [] },
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
    answer: () => undefined,
    artifactsByMessage: new Map(),
    block: null,
    current: null,
    decided: () => undefined,
    deferAll: () => undefined,
    dismiss: async () => undefined,
    draft: "",
    followUpDecision: () => undefined,
    latestUserSequence: 1,
    onDraftChange: () => undefined,
    proposalsQuery: { data: { results: [] } },
    providerId: "",
    providers: [],
    providersReady: true,
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

afterEach(cleanup);

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
