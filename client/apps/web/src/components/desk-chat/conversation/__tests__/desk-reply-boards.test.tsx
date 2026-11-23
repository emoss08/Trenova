import {
  ReplyBoardsProvider,
  type ReplyBoard,
  type ReplyBoardRenderer,
  type ReplyBoardsProps,
} from "@/components/assistant/artifact-opener";
import type { ThreadEntry } from "@/components/assistant/thread-view";
import { DESK_ACTION_ROW } from "@/components/desk-chat/conversation/desk-message-actions";
import { DeskReply } from "@/components/desk-chat/conversation/desk-turns";
import type { AssistantArtifact, AssistantMessage } from "@/types/assistant";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { lazy, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(cleanup);

const REPLY =
  "There are 6 late loads, the furthest 3h 10m behind ([Six late loads](artifact:art_inline)).";

const entry = {
  kind: "assistant",
  message: {
    id: "amsg_1",
    threadId: "athr_1",
    role: "Assistant",
    content: REPLY,
    createdAt: 1_793_114_000,
  } as AssistantMessage,
  tools: [],
} satisfies Extract<ThreadEntry, { kind: "assistant" }>;

const artifact = {
  id: "art_inline",
  threadId: "athr_1",
  kind: "board",
  title: "Six late loads",
  payload: { place: "conversation" },
  sourceToolCallId: "call_inline",
  lineageId: "",
  lineageSeq: 1,
} as unknown as AssistantArtifact;

const BOARDS: ReplyBoard[] = [{ artifact, tool: "show_board", spec: null }];

const drawn = vi.fn();

function StubBoards(props: ReplyBoardsProps) {
  drawn(props);
  return <p>Board {props.boards.map((board) => board.artifact.id).join(", ")}</p>;
}

const RENDERER: ReplyBoardRenderer = { Boards: StubBoards, Pending: () => null };

function draw(boards: readonly ReplyBoard[] | undefined, wrap: (node: ReactNode) => ReactNode) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        {wrap(
          <DeskReply
            entry={entry}
            steps={[]}
            threadArtifacts={[artifact]}
            boards={boards}
            arrives
            onOpenArtifact={() => {}}
          />,
        )}
      </TestRouter>
    </QueryClientProvider>,
  );
}

/*
Only the Desk's conversation supplies a way to draw a board. A reply there
whose turn placed a board in the conversation draws the board instead of its
words; on every other surface, and for a reply with no board, it draws its
words, which already say the answer and link the board.
*/
describe("DeskReply and the boards its turn placed in the conversation", () => {
  it("draws the board in place of its words where the surface draws boards", () => {
    drawn.mockClear();
    draw(BOARDS, (node) => <ReplyBoardsProvider renderer={RENDERER}>{node}</ReplyBoardsProvider>);

    expect(screen.getByText("Board art_inline")).toBeInTheDocument();
    expect(screen.queryByText(/There are 6 late loads/u)).not.toBeInTheDocument();
    const props = drawn.mock.calls.at(-1)?.[0] as ReplyBoardsProps;
    expect(props.answeredAt).toBe(1_793_114_000);
    expect(props.arrives).toBe(true);
    expect(props.boards).toBe(BOARDS);
  });

  it("hands the board its words to draw when the board cannot be read", () => {
    drawn.mockClear();
    draw(BOARDS, (node) => <ReplyBoardsProvider renderer={RENDERER}>{node}</ReplyBoardsProvider>);

    const props = drawn.mock.calls.at(-1)?.[0] as ReplyBoardsProps;
    render(<div data-testid="fallback">{props.fallback}</div>);
    expect(screen.getByTestId("fallback")).toHaveTextContent(
      "There are 6 late loads, the furthest 3h 10m behind (Six late loads).",
    );
  });

  it("draws its words on a surface that draws no boards", () => {
    draw(BOARDS, (node) => node);

    expect(screen.queryByText(/^Board /u)).not.toBeInTheDocument();
    expect(screen.getByText(/There are 6 late loads/u)).toBeInTheDocument();
  });

  it("draws its words when its turn placed no board in the conversation", () => {
    draw(undefined, (node) => (
      <ReplyBoardsProvider renderer={RENDERER}>{node}</ReplyBoardsProvider>
    ));

    expect(screen.queryByText(/^Board /u)).not.toBeInTheDocument();
    expect(screen.getByText(/There are 6 late loads/u)).toBeInTheDocument();
  });
});

function Reply({ arrives, renderer }: { arrives: boolean; renderer: ReplyBoardRenderer }) {
  return (
    <ReplyBoardsProvider renderer={renderer}>
      <DeskReply
        entry={entry}
        steps={[]}
        threadArtifacts={[artifact]}
        boards={BOARDS}
        arrives={arrives}
        onOpenArtifact={() => {}}
      />
    </ReplyBoardsProvider>
  );
}

/*
A reply read from history never arrives. Whether a reply arrives is settled
when the reply mounts: the thread marks the newest reply once its history has
settled, a frame after the history is drawn, and the board's code may load
later still. Neither may turn a reply that was already on screen into one
that lands.
*/
describe("DeskReply's arrival", () => {
  it("keeps a reply that mounted without arriving from arriving when the thread later says it does", () => {
    drawn.mockClear();
    const queryClient = new QueryClient();
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <Reply arrives={false} renderer={RENDERER} />
      </QueryClientProvider>,
    );
    rerender(
      <QueryClientProvider client={queryClient}>
        <Reply arrives renderer={RENDERER} />
      </QueryClientProvider>,
    );

    expect(drawn.mock.calls.length).toBeGreaterThan(1);
    expect(drawn.mock.calls.every(([props]) => (props as ReplyBoardsProps).arrives === false)).toBe(
      true,
    );
  });

  it("gives a board whose code loads after the reply mounted the answer the reply had at mount", async () => {
    drawn.mockClear();
    let load: () => void = () => {};
    const loaded = new Promise<void>((resolve) => {
      load = resolve;
    });
    const LazyStub = lazy(async () => {
      await loaded;
      return { default: StubBoards };
    });
    const renderer: ReplyBoardRenderer = {
      Boards: (props) => (
        <LazyComponent fallback={null}>
          <LazyStub {...props} />
        </LazyComponent>
      ),
      Pending: () => null,
    };
    const queryClient = new QueryClient();
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <Reply arrives={false} renderer={renderer} />
      </QueryClientProvider>,
    );
    rerender(
      <QueryClientProvider client={queryClient}>
        <Reply arrives renderer={renderer} />
      </QueryClientProvider>,
    );
    expect(drawn).not.toHaveBeenCalled();

    await act(async () => {
      load();
      await loaded;
    });

    expect(await screen.findByText("Board art_inline")).toBeInTheDocument();
    expect(drawn.mock.calls.every(([props]) => (props as ReplyBoardsProps).arrives === false)).toBe(
      true,
    );
  });

  it("lets a reply that mounted arriving arrive", () => {
    drawn.mockClear();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Reply arrives renderer={RENDERER} />
      </QueryClientProvider>,
    );

    expect(drawn).toHaveBeenLastCalledWith(expect.objectContaining({ arrives: true }));
  });
});

/*
An answer that took several model steps is saved as several replies. Each used
to carry its own bar of actions, hidden until hovered but a bar's height all the
same, so one question read as several replies with gaps between them.
*/
describe("the actions under a reply", () => {
  function step(tail: boolean | undefined) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <QueryClientProvider client={queryClient}>
        <TestRouter>
          <DeskReply
            entry={entry}
            steps={[]}
            threadArtifacts={[artifact]}
            tail={tail}
            onOpenArtifact={() => {}}
          />
        </TestRouter>
      </QueryClientProvider>,
    );
  }

  it("are drawn under the last step of an answer", () => {
    step(true);

    expect(screen.getByRole("button", { name: "Copy reply" })).toBeInTheDocument();
  });

  it("sit in the row box a board being read keeps for them", () => {
    step(true);

    expect(screen.getByRole("button", { name: "Copy reply" }).parentElement).toHaveClass(
      "dk-acts",
      ...DESK_ACTION_ROW.split(" "),
    );
  });

  it("are left off a step that another step follows, and its words still stand", () => {
    step(false);

    expect(screen.queryByRole("button", { name: "Copy reply" })).not.toBeInTheDocument();
    expect(screen.getByText(/There are 6 late loads/u)).toBeInTheDocument();
  });

  it("are drawn when the thread says nothing about steps, as a reply drawn on its own is", () => {
    step(undefined);

    expect(screen.getByRole("button", { name: "Copy reply" })).toBeInTheDocument();
  });
});

/*
A reply drawn as a board keeps its actions until the board is drawn: a bar
under the board's skeleton, before the board lands, read as the reply ending
and then growing. The board says when it is drawn, or could not be read.
*/
describe("the actions under a reply drawn as boards", () => {
  function LandingBoards(props: ReplyBoardsProps) {
    return (
      <button type="button" onClick={props.onDrawn}>
        Land the board
      </button>
    );
  }

  const landing: ReplyBoardRenderer = { Boards: LandingBoards, Pending: () => null };

  it("are not drawn while the board is read, and come with the board", () => {
    draw(BOARDS, (node) => <ReplyBoardsProvider renderer={landing}>{node}</ReplyBoardsProvider>);

    expect(screen.queryByRole("button", { name: "Copy reply" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Land the board" }));

    expect(screen.getByRole("button", { name: "Copy reply" })).toBeInTheDocument();
  });

  it("are drawn at once under a reply in words where the surface draws boards", () => {
    draw(undefined, (node) => <ReplyBoardsProvider renderer={landing}>{node}</ReplyBoardsProvider>);

    expect(screen.getByRole("button", { name: "Copy reply" })).toBeInTheDocument();
  });
});
