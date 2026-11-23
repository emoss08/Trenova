import { ArtifactOpenerProvider } from "@/components/assistant/artifact-opener";
import type { BoardCallTool } from "@/lib/board-place";
import type { DeskBoardCard } from "@/lib/graphql/desk-board-card";
import { useDeskStore } from "@/stores/desk-store";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { cardTime } from "../../../record-card/record-card-values";
import { DESK_ACTION_ROW } from "@/components/desk-chat/conversation/desk-message-actions";
import { DeskPendingBoard, DeskReplyBoards } from "../board-reply";
import {
  ANSWERED_AT,
  CONVERSATION_SPEC,
  conversationBoard,
  count,
  segment,
  summaryBoard,
} from "./fixtures";

const fetchers = vi.hoisted(() => ({ card: vi.fn() }));

vi.mock("@/lib/graphql/desk-board-card", () => ({ fetchDeskBoardCard: fetchers.card }));

/** A show_answer call's arguments, in that tool's shape (`runtimetools/answer.go`). */
const ANSWER_ARGS = {
  title: "Who the late loads wait on",
  answer: "There are {count} late loads, the furthest {largest} behind.",
  measure: "lateness",
  groups: [
    {
      title: "Waiting on one driver's hours",
      records: ["shp_1", "shp_2", "shp_3", "shp_4"],
      note: "Marcus Bell is out of hours, with {count_1} loads waiting on him.",
    },
    { title: "Late at the dock", records: ["shp_5", "shp_6"] },
  ],
};

const REPLY_WORDS =
  "There are 6 late loads, the furthest 3h 10m behind, with — owed. Four wait on one driver's hours.";

/** Brings every reply the test drew within reach of the viewport. */
function scrollIntoReach() {
  act(() => {
    const targets = FakeIntersectionObserver.instances.flatMap((observer) =>
      Array.from(observer.targets),
    );
    for (const target of targets) FakeIntersectionObserver.show(target);
  });
}

function draw(
  card: DeskBoardCard | Promise<DeskBoardCard>,
  {
    onOpen = vi.fn(),
    spec = CONVERSATION_SPEC,
    tool = "show_board",
    inReach = true,
    onDrawn,
    opens = true,
  }: {
    onOpen?: Mock<(id: string) => void>;
    spec?: Record<string, unknown> | null;
    tool?: BoardCallTool;
    inReach?: boolean;
    onDrawn?: () => void;
    /** The surface opens boards beside the conversation, as the Desk's does. */
    opens?: boolean;
  } = {},
) {
  if (card instanceof Promise) {
    fetchers.card.mockReturnValue(card);
  } else {
    fetchers.card.mockResolvedValue(card);
  }
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const boards = (
    <DeskReplyBoards
      boards={[{ artifact: summaryBoard(), tool, spec }]}
      fallback={<p data-testid="reply-words">{REPLY_WORDS}</p>}
      answeredAt={ANSWERED_AT}
      arrives={false}
      onDrawn={onDrawn}
    />
  );
  render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        {opens ? <ArtifactOpenerProvider onOpen={onOpen}>{boards}</ArtifactOpenerProvider> : boards}
      </TestRouter>
    </QueryClientProvider>,
  );
  if (inReach) scrollIntoReach();
  return { onOpen };
}

const OPENED = {
  artifactId: "art_inline",
  entity: "shipment",
  recordId: "shp_2",
  order: ["shp_1", "shp_2"],
};

beforeEach(() => {
  useDeskStore.setState({ artifactRecord: null });
  installIntersectionObserver();
  seedPermissions({ [Resource.Shipment]: Operation.Read });
  fetchers.card.mockReset();
});

afterEach(() => {
  cleanup();
  clearPermissions();
  vi.unstubAllGlobals();
});

/*
A board placed in the conversation is the reply: its verdict as the reply's
words, its blocks as exhibits under them, and a foot that opens the board
beside the conversation. The reply's own text is what every other surface
draws, and what the Desk draws when the board cannot be read.
*/
describe("DeskReplyBoards", () => {
  it("draws the verdict as the reply's words and the blocks as exhibits, not the reply's text", async () => {
    draw(conversationBoard());

    const board = await screen.findByRole("article", {
      name: "Six late loads on the Dallas lanes",
    });
    expect(screen.queryByTestId("reply-words")).not.toBeInTheDocument();
    expect(within(board).queryByRole("heading")).not.toBeInTheDocument();
    expect(board.querySelector(".dk-prose")).toHaveTextContent(
      "There are 6 late loads, the furthest 3h 10m behind, with an amount you may not read owed. Four wait on one driver's hours1.",
    );
    expect(
      within(board).getByRole("region", { name: "The late loads, furthest behind first" }),
    ).toBeInTheDocument();
    expect(fetchers.card).toHaveBeenCalledWith("athr_1", "art_inline", expect.anything());
  });

  it("cuts a list to six rows and says how many more the board holds, opening the board", async () => {
    const { onOpen } = draw(conversationBoard());

    const list = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    const rows = list.querySelectorAll(".dk-bd-rows > .dk-bd-row");
    expect(rows).toHaveLength(6);
    expect(rows[3]).toHaveClass("dk-gone");
    expect(rows[3]).toHaveTextContent("This shipment could not be read");

    fireEvent.click(within(list).getByRole("button", { name: "2 more on the board" }));
    expect(onOpen).toHaveBeenCalledWith("art_inline");
  });

  it("opens the pane at the clicked row's own record, stepping through the block's whole list", async () => {
    const { onOpen } = draw(conversationBoard());

    fireEvent.click(await screen.findByRole("button", { name: /S-24110/u }));

    expect(onOpen).toHaveBeenCalledWith("art_inline");
    // The list draws six rows, but the record steps through every row of the block a person may read.
    const shown = conversationBoard().blocks[0].rows;
    expect(useDeskStore.getState().artifactRecord).toEqual({
      artifactId: "art_inline",
      entity: "shipment",
      recordId: shown[0].id,
      order: shown.filter((row) => row.readable).map((row) => row.id),
    });
  });

  it("opens the board itself, not a record, from the foot", async () => {
    useDeskStore.getState().openArtifactRecord(OPENED);
    const { onOpen } = draw(conversationBoard());

    fireEvent.click(await screen.findByRole("button", { name: "Open the board" }));

    expect(onOpen).toHaveBeenLastCalledWith("art_inline");
    expect(useDeskStore.getState().artifactRecord).toBeNull();
  });

  it("opens the board itself, not a record, from the count of rows it holds beyond the list", async () => {
    useDeskStore.getState().openArtifactRecord(OPENED);
    const { onOpen } = draw(conversationBoard());

    fireEvent.click(await screen.findByRole("button", { name: "2 more on the board" }));

    expect(onOpen).toHaveBeenLastCalledWith("art_inline");
    expect(useDeskStore.getState().artifactRecord).toBeNull();
  });

  // jsdom lays nothing out, so the class that closes the gap is the most that can be asserted.
  it("pulls an actions block that follows an exhibit up under it, and not one set half width", async () => {
    draw(conversationBoard());
    const full = (await screen.findByRole("button", { name: "Ask the Memphis dock" })).closest(
      ".-mt-7",
    );
    expect(full).not.toBeNull();

    cleanup();
    const card = conversationBoard();
    const [records, actions] = card.blocks;
    card.blocks = [
      records,
      { ...actions, half: true },
      {
        ...actions,
        half: true,
        actions: [{ label: "Ask the Dallas dock", ask: "Ask the Dallas dock", records: [] }],
      },
    ];
    draw(card);
    const half = await screen.findByRole("button", { name: "Ask the Memphis dock" });
    expect(half.closest(".-mt-7")).toBeNull();
  });

  it("links a row to its record's page for a reader without its card", async () => {
    seedPermissions({});
    draw(conversationBoard());

    const list = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    expect(within(list).queryByRole("button", { name: /S-24110/u })).not.toBeInTheDocument();
    expect(
      within(list)
        .getByRole("link", { name: /S-24110/u })
        .getAttribute("href"),
    ).toContain("shp_1");
  });

  it("queues an action's request as the person's own message, as the board's actions do", async () => {
    draw(conversationBoard());

    fireEvent.click(await screen.findByRole("button", { name: "Ask the Memphis dock" }));
    expect(useDeskStore.getState().takeAsk("athr_1")).toBe(
      "Ask the Memphis dock for an unload time",
    );
  });

  it("lights the exhibit a cited phrase cites while the phrase is pointed at", async () => {
    draw(conversationBoard());

    const list = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    const phrase = screen.getByText("Four wait on one driver's hours");
    fireEvent.mouseEnter(phrase.parentElement as HTMLElement);
    expect(list).toHaveClass("dk-bd-lit");
    fireEvent.mouseLeave(phrase.parentElement as HTMLElement);
    expect(list).not.toHaveClass("dk-bd-lit");
  });

  it("reads nothing until the reply comes within reach of the viewport", async () => {
    draw(conversationBoard(), { inReach: false });

    expect(screen.getByLabelText("Reading the board…")).toHaveAttribute("aria-busy", "true");
    expect(fetchers.card).not.toHaveBeenCalled();
    expect(FakeIntersectionObserver.instances[0]?.options.rootMargin).toBe("600px 0px");

    scrollIntoReach();
    expect(
      await screen.findByRole("article", { name: "Six late loads on the Dallas lanes" }),
    ).toBeInTheDocument();
    expect(fetchers.card).toHaveBeenCalledTimes(1);
  });

  it("draws the words as written and every block's shape while the board is read", () => {
    draw(new Promise<DeskBoardCard>(() => {}));

    const waiting = screen.getByLabelText("Reading the board…");
    expect(waiting).toHaveAttribute("aria-busy", "true");
    expect(waiting.querySelector(".dk-prose")).toHaveTextContent(
      "There are late loads, the furthest behind, with owed. Four wait on one driver's hours1.",
    );
    expect(within(waiting).getByText("The late loads, furthest behind first")).toBeInTheDocument();
    expect(screen.queryByTestId("reply-words")).not.toBeInTheDocument();
  });

  it("draws the reply's own words while the board is read when the call's layout is not in view", () => {
    draw(new Promise<DeskBoardCard>(() => {}), { spec: null });

    expect(screen.getByTestId("reply-words")).toHaveTextContent(REPLY_WORDS);
    expect(screen.getByLabelText("Reading the board…")).toHaveAttribute("aria-busy", "true");
  });

  it("draws the reply's words, a notice and Try again when the board cannot be read", async () => {
    fetchers.card.mockRejectedValue(new Error("unavailable"));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <TestRouter>
          <DeskReplyBoards
            boards={[{ artifact: summaryBoard(), tool: "show_board", spec: CONVERSATION_SPEC }]}
            fallback={<p data-testid="reply-words">{REPLY_WORDS}</p>}
            answeredAt={ANSWERED_AT}
            arrives={false}
          />
        </TestRouter>
      </QueryClientProvider>,
    );

    scrollIntoReach();

    const notice = await screen.findByRole("region", { name: "The exhibits could not be read" });
    expect(screen.getByTestId("reply-words")).toHaveTextContent(REPLY_WORDS);
    expect(notice).toHaveTextContent("The late loads, furthest behind first");
    expect(notice).toHaveTextContent(
      `The words above keep what was read at ${cardTime(ANSWERED_AT)}.`,
    );
    expect(screen.queryByRole("button", { name: "Open the board" })).not.toBeInTheDocument();

    fetchers.card.mockResolvedValue(conversationBoard());
    fireEvent.click(within(notice).getByRole("button", { name: "Try again" }));
    expect(
      await screen.findByRole("article", { name: "Six late loads on the Dallas lanes" }),
    ).toBeInTheDocument();
    await waitFor(() => expect(fetchers.card).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId("reply-words")).not.toBeInTheDocument();
  });

  it("draws an exhibit's caption under its tab with its readings, and nothing where it has none", async () => {
    const card = conversationBoard();
    card.blocks[0].caption = [
      segment("Marcus Bell is out of hours, with "),
      segment("4", { reading: "late" }),
      segment(" loads waiting on him."),
    ];
    card.readings[0] = { ...card.readings[0], value: count(4) };
    draw(card);

    const list = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    const caption = list.querySelector("p") as HTMLElement;
    expect(caption).toHaveTextContent("Marcus Bell is out of hours, with 4 loads waiting on him.");
    expect(within(caption).getByRole("button", { name: "Late now: 4" })).toBeInTheDocument();
    expect(list.querySelector(".dk-bd-tab + p")).toBe(caption);

    cleanup();
    draw(conversationBoard());
    const plain = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    expect(plain.querySelector("p")).toBeNull();
  });

  it("lays a show_answer board out from its call while it is read: its groups side by side, each note with blanks", () => {
    draw(new Promise<DeskBoardCard>(() => {}), { tool: "show_answer", spec: ANSWER_ARGS });

    const waiting = screen.getByLabelText("Reading the board…");
    expect(waiting.querySelector(".dk-prose")).toHaveTextContent(
      "There are late loads, the furthest behind.",
    );
    const exhibits = [...waiting.querySelectorAll("section.dk-bd-ex")];
    expect(exhibits.map((exhibit) => exhibit.querySelector(".dk-bd-tab")?.textContent)).toEqual([
      "1Waiting on one driver's hours",
      "2Late at the dock",
    ]);
    expect(exhibits.map((exhibit) => exhibit.classList.contains("dk-bd-half"))).toEqual([
      true,
      true,
    ]);
    expect(exhibits[0].querySelector("p")).toHaveTextContent(
      "Marcus Bell is out of hours, with loads waiting on him.",
    );
    expect(exhibits[0].querySelectorAll("p .dk-bd-blank")).toHaveLength(1);
    expect(exhibits[1].querySelector("p")).toBeNull();
  });

  it("draws the skeleton the running turn drew while a saved board with no verdict is read, never the reply's words", () => {
    const { verdict: _verdict, ...unworded } = CONVERSATION_SPEC;
    const { container: running } = render(
      <ArtifactOpenerProvider onOpen={vi.fn()}>
        <DeskPendingBoard tool="show_board" spec={unworded} />
      </ArtifactOpenerProvider>,
    );
    const drawnLive = running.innerHTML;
    cleanup();

    draw(new Promise<DeskBoardCard>(() => {}), { spec: unworded });

    expect(screen.queryByTestId("reply-words")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Reading the board…").outerHTML).toBe(drawnLive);
  });

  it("tells the reply its boards are drawn once the board is read, and not while it is", async () => {
    let land: (card: DeskBoardCard) => void = () => {};
    const onDrawn = vi.fn();
    draw(
      new Promise<DeskBoardCard>((resolve) => {
        land = resolve;
      }),
      { onDrawn },
    );
    expect(onDrawn).not.toHaveBeenCalled();

    await act(async () => land(conversationBoard()));

    expect(
      await screen.findByRole("article", { name: "Six late loads on the Dallas lanes" }),
    ).toBeInTheDocument();
    expect(onDrawn).toHaveBeenCalled();
  });

  it("tells the reply its boards are drawn when the board cannot be read, as its words then stand", async () => {
    const onDrawn = vi.fn();
    const failed = Promise.reject(new Error("unavailable"));
    failed.catch(() => undefined);
    draw(failed, { onDrawn });

    await screen.findByRole("region", { name: "The exhibits could not be read" });
    expect(onDrawn).toHaveBeenCalled();
  });

  it("does not tell the reply its boards are drawn before a board out of reach is read", () => {
    const onDrawn = vi.fn();
    draw(conversationBoard(), { inReach: false, onDrawn });

    expect(onDrawn).not.toHaveBeenCalled();
  });

  // jsdom lays nothing out, so the skeleton's height is held by sharing the board's own parts and
  // boxes (SourceLine, the row box, ReplyFoot, DESK_ACTION_ROW); these tests assert that sharing.
  it("keeps each exhibit's source line while it is read, in the line the drawn exhibit uses", async () => {
    let land: (card: DeskBoardCard) => void = () => {};
    draw(
      new Promise<DeskBoardCard>((resolve) => {
        land = resolve;
      }),
    );

    const waiting = screen.getByLabelText("Reading the board…");
    const sketched = waiting.querySelector("section.dk-bd-ex .dk-bd-src");
    expect(
      [...(sketched?.children ?? [])].map(
        (part) => part.querySelectorAll(".dk-bd-wait-line").length,
      ),
    ).toEqual([1, 1]);
    expect(sketched?.textContent).toBe("\u200b\u200b");

    await act(async () => land(conversationBoard()));
    const drawn = await screen.findByRole("region", {
      name: "The late loads, furthest behind first",
    });
    expect(drawn.querySelector(".dk-bd-src")?.children).toHaveLength(2);
  });

  it("draws each listed record in the drawn row's own box, cut where the conversation cuts the list", () => {
    const eight = Array.from({ length: 8 }, (_, index) => `shp_${index + 1}`);
    draw(new Promise<DeskBoardCard>(() => {}), {
      tool: "show_answer",
      spec: {
        ...ANSWER_ARGS,
        groups: [ANSWER_ARGS.groups[1], { title: "Every late load", records: eight }],
      },
    });

    const exhibits = [
      ...screen.getByLabelText("Reading the board…").querySelectorAll("section.dk-bd-ex"),
    ];
    const rows = exhibits.map((exhibit) => exhibit.querySelectorAll(".dk-bd-rows > .dk-bd-row"));
    expect(rows.map((list) => list.length)).toEqual([2, 6]);
    expect([...rows[0][0].querySelectorAll(".dk-bd-who > *")].map((line) => line.tagName)).toEqual([
      "B",
      "SPAN",
    ]);
    expect(within(exhibits[1] as HTMLElement).getByText("2 more on the board")).toBeInTheDocument();
  });

  it("holds the foot under the last exhibit while it is read, only where the surface opens boards", () => {
    draw(new Promise<DeskBoardCard>(() => {}), { tool: "show_answer", spec: ANSWER_ARGS });

    const exhibits = [
      ...screen.getByLabelText("Reading the board…").querySelectorAll("section.dk-bd-ex"),
    ];
    expect(
      exhibits.map((exhibit) => exhibit.textContent?.includes("From the board for this answer")),
    ).toEqual([false, true]);
    expect(exhibits[1].querySelector(".dk-bd-rows")?.lastElementChild).toHaveTextContent(
      "From the board for this answer",
    );
    expect(screen.queryByRole("button", { name: "Open the board" })).not.toBeInTheDocument();

    cleanup();
    draw(new Promise<DeskBoardCard>(() => {}), {
      tool: "show_answer",
      spec: ANSWER_ARGS,
      opens: false,
    });
    expect(screen.queryByText("From the board for this answer")).not.toBeInTheDocument();
  });

  it("says the board is being read in the row the reply's actions take, last under it", () => {
    draw(new Promise<DeskBoardCard>(() => {}));

    const waiting = screen.getByLabelText("Reading the board…");
    const row = within(waiting).getByText("Reading what the answer rests on…");
    expect(row).toBe(waiting.lastElementChild);
    expect(row).toHaveClass(...DESK_ACTION_ROW.split(" "));
  });

  it("draws a blank for each of one record's own figures while a show_answer board is read, and no brace text", () => {
    const { container } = render(
      <DeskPendingBoard
        tool="show_answer"
        spec={{
          title: "SEED-SHP-010 against SEED-SHP-008",
          answer:
            "SEED-SHP-010 brings in {revenue:SEED-SHP-010} and SEED-SHP-008 brings in {revenue:SEED-SHP-008}.",
          groups: [
            { title: "SEED-SHP-010", records: ["SEED-SHP-010", "SEED-SHP-011"] },
            { title: "SEED-SHP-008", records: ["SEED-SHP-008"] },
          ],
        }}
      />,
    );

    expect(container.querySelectorAll(".dk-bd-blank")).toHaveLength(2);
    expect(container).not.toHaveTextContent("{");
    expect(container).toHaveTextContent("SEED-SHP-010 brings in and SEED-SHP-008 brings in .");
  });

  it("draws a blank for each own figure written by number over records passed by id, and no brace text", () => {
    const { container } = render(
      <DeskPendingBoard
        tool="show_answer"
        spec={{
          title: "SEED-SHP-010 against SEED-SHP-008",
          answer:
            "SEED-SHP-010 is {lateness:SEED-SHP-010} behind and brings in {revenue:SEED-SHP-010}, against {lateness:SEED-SHP-008} and {revenue:SEED-SHP-008}.",
          groups: [
            {
              title: "A",
              records: ["shp_01JTEN00000000000000000001", "shp_01JTEN00000000000000000002"],
            },
            { title: "B", records: ["shp_01JEIGHT000000000000000001"] },
          ],
        }}
      />,
    );

    expect(container.querySelectorAll(".dk-bd-blank")).toHaveLength(4);
    expect(container).not.toHaveTextContent("{");
  });

  it("draws a running show_answer call's board as its skeleton", () => {
    render(<DeskPendingBoard tool="show_answer" spec={ANSWER_ARGS} />);

    expect(screen.getByLabelText("Reading the board…")).toHaveAttribute("aria-busy", "true");
    expect(screen.getByText("Waiting on one driver's hours")).toBeInTheDocument();
    expect(screen.getByText("Late at the dock")).toBeInTheDocument();
  });
});
