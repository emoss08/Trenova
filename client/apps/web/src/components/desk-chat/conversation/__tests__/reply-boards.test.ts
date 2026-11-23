import type { ThreadEntry, ToolExchange } from "@/components/assistant/thread-view";
import type { TurnSegment } from "@/components/assistant/turn-stream";
import type {
  AssistantArtifact,
  AssistantArtifactEvent,
  AssistantMessage,
} from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  linkedArtifactIds,
  pendingReplyBoard,
  replyBoardsByEntry,
  workingSteps,
} from "../reply-boards";

/** A show_board call that asked for the conversation, as it must to be placed there. */
const SPEC = {
  title: "Six late loads",
  place: "conversation",
  verdict: "There are {late} late loads.",
  blocks: [{ type: "records", title: "The late loads" }],
};

/** A show_answer call, in that tool's shape (`runtimetools/answer.go`). */
const ANSWER = {
  title: "Late loads on the Dallas lanes",
  answer: "There are {count} late loads, the furthest {largest} behind.",
  measure: "lateness",
  groups: [{ title: "Late at the dock", records: ["shp_1", "shp_2"] }],
};

function message(id: string, content = ""): AssistantMessage {
  return { id, threadId: "athr_1", content, role: "Assistant", createdAt: 1 } as AssistantMessage;
}

function user(id: string): ThreadEntry {
  return { kind: "user", message: { ...message(id, "Which loads are late?"), role: "User" } };
}

function showBoard(id: string, args: Record<string, unknown> | null = SPEC): ToolExchange {
  return { call: { id, name: "show_board", arguments: args }, result: null };
}

function showAnswer(id: string, args: Record<string, unknown> | null = ANSWER): ToolExchange {
  return { call: { id, name: "show_answer", arguments: args }, result: null };
}

function reply(id: string, content: string, tools: ToolExchange[] = []): ThreadEntry {
  return { kind: "assistant", message: message(id, content), tools };
}

/** A board as a thread's artifact summary lists it: its place kept, its layout left out. */
function board(id: string, fields: Partial<AssistantArtifact> = {}): AssistantArtifact {
  return {
    id,
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "board",
    status: "Ready",
    title: "Six late loads",
    payload: { place: "conversation" },
    sourceToolCallId: `call_${id}`,
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: id,
    turn: "",
    createdAt: 1,
    updatedAt: 1,
    ...fields,
  } as AssistantArtifact;
}

const link = (id: string) => `There are 6 late loads ([Six late loads](artifact:${id})).`;

/*
The reply the server writes for a board in the conversation links that board,
and ends the turn. That reply draws the boards its turn placed there, in the
order they were shown, while each is still the newest of its lineage. A board
beside the conversation, or a reply that links none, draws nothing new.
*/
describe("replyBoardsByEntry", () => {
  it("gives the turn's conversation boards to the reply that links them, with the call's layout", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    const boards = replyBoardsByEntry(entries, [board("art_a")]);

    expect([...boards.keys()]).toEqual(["a1"]);
    expect(boards.get("a1")?.map((item) => [item.artifact.id, item.spec])).toEqual([
      ["art_a", SPEC],
    ]);
  });

  it("draws every conversation board of the turn in the order it was shown", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_b"), showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    const boards = replyBoardsByEntry(entries, [board("art_a"), board("art_b")]);

    expect(boards.get("a1")?.map((item) => item.artifact.id)).toEqual(["art_b", "art_a"]);
  });

  it("leaves a board placed beside the conversation, and one stored before boards had a place", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
      user("u2"),
      reply("c2", "", [showBoard("call_art_b")]),
      reply("a2", link("art_b")),
    ];

    const boards = replyBoardsByEntry(entries, [
      board("art_a", { payload: { place: "beside" } }),
      board("art_b", { payload: { board: { title: "Old" } } }),
    ]);

    expect(boards.size).toBe(0);
  });

  it("reads the place from a full payload's board as well as a summary's", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    const boards = replyBoardsByEntry(entries, [
      board("art_a", { payload: { board: { place: "conversation" }, author: "Dispatch desk" } }),
    ]);

    expect(boards.get("a1")?.map((item) => item.artifact.id)).toEqual(["art_a"]);
  });

  it("draws only the newest version: a revised board's reply draws it, the earlier reply its words", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
      user("u2"),
      reply("c2", "", [showBoard("call_art_a2")]),
      reply("a2", link("art_a2")),
    ];

    const boards = replyBoardsByEntry(entries, [
      board("art_a"),
      board("art_a2", { lineageId: "art_a", lineageSeq: 2 }),
    ]);

    expect([...boards.keys()]).toEqual(["a2"]);
    expect(boards.get("a2")?.map((item) => item.artifact.id)).toEqual(["art_a2"]);
  });

  it("gives nothing to a reply that does not link the board, nor to a later turn that cites it", () => {
    const entries = [
      user("u1"),
      reply("c1", "Let me look.", [showBoard("call_art_a")]),
      reply("a1", "Here is what I found."),
      user("u2"),
      reply("a2", link("art_a")),
    ];

    expect(replyBoardsByEntry(entries, [board("art_a")]).size).toBe(0);
  });

  it("draws nothing for a reply linking a board its turn did not show, and keeps the layout null when the call is not in view", () => {
    const elsewhere = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a")]),
      reply("a1", link("art_z")),
    ];
    expect(replyBoardsByEntry(elsewhere, [board("art_a"), board("art_z")]).size).toBe(0);

    const orphan = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_a", {})]),
      reply("a1", link("art_a")),
    ];
    expect(replyBoardsByEntry(orphan, [board("art_a")]).get("a1")?.[0].spec).toBeNull();
  });
});

/*
The saved reply reaches the thread before the artifact list catches up with
the board it links. For that moment the reply draws the board it links rather
than its words, provided the turn has one show_board call the list does not
hold yet, that call asked for a board the conversation takes, and the replies
link one artifact the list does not hold.
*/
describe("replyBoardsByEntry before the artifact list holds the board", () => {
  it("draws the board the saved reply links, read by its id, with the call's layout", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_new")]),
      reply("a1", link("art_new")),
    ];

    const drawn = replyBoardsByEntry(entries, [board("art_old")]).get("a1");

    expect(
      drawn?.map((item) => [item.artifact.id, item.artifact.threadId, item.artifact.title]),
    ).toEqual([["art_new", "athr_1", "Six late loads"]]);
    expect(drawn?.[0].spec).toEqual(SPEC);
  });

  it("draws nothing early for a call that asked to be beside, or when it cannot tell which board is the call's", () => {
    const beside = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_new", { ...SPEC, place: "beside" })]),
      reply("a1", link("art_new")),
    ];
    expect(replyBoardsByEntry(beside, []).size).toBe(0);

    const two = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_x"), showBoard("call_art_y")]),
      reply("a1", link("art_x")),
    ];
    expect(replyBoardsByEntry(two, []).size).toBe(0);
  });

  it("never stands in for a show_board call that failed, even when the reply links an artifact the list does not hold", () => {
    const failed: ToolExchange = {
      ...showBoard("call_art_new"),
      result: { ...message("r1", 'Tool "show_board" was not run.'), toolFailed: true },
    };
    const entries = [
      user("u1"),
      reply("c1", "", [failed]),
      reply("a1", "The late loads are in [Shipments](artifact:art_table)."),
    ];

    expect(replyBoardsByEntry(entries, []).size).toBe(0);
  });
});

/*
Two tools make a board in the conversation. show_answer always does; the
server builds its board in the conversation. show_board is the larger page and
is beside the conversation unless its call asked for the conversation and
holds at most four blocks.
*/
describe("which calls make a board in the conversation", () => {
  const running = (name: string, args: Record<string, unknown>) =>
    pendingReplyBoard([call({ name, arguments: args })], NOTHING);
  const five = Array.from({ length: 5 }, () => ({ type: "records" }));

  it("takes every show_answer call, whatever it holds", () => {
    expect(running("show_answer", ANSWER)).toEqual({
      tool: "show_answer",
      spec: ANSWER,
      kept: "",
      working: "",
      saved: false,
    });
    expect(running("show_answer", { ...ANSWER, place: "beside" })?.tool).toBe("show_answer");
  });

  it("takes a show_board call only when it asked for the conversation and holds at most four blocks", () => {
    expect(running("show_board", SPEC)?.tool).toBe("show_board");
    const { place: _place, ...unasked } = SPEC;
    expect(running("show_board", unasked)).toBeNull();
    expect(running("show_board", { ...SPEC, blocks: five })).toBeNull();
  });

  it("gives a saved show_answer board to its reply with the tool and the call's arguments", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showAnswer("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    expect(
      replyBoardsByEntry(entries, [board("art_a")])
        .get("a1")
        ?.map((item) => [item.artifact.id, item.tool, item.spec]),
    ).toEqual([["art_a", "show_answer", ANSWER]]);
  });

  it("stands in for an unlisted show_answer board, but not for a show_board call that did not ask for the conversation", () => {
    const answered = [
      user("u1"),
      reply("c1", "", [showAnswer("call_art_new")]),
      reply("a1", link("art_new")),
    ];
    expect(
      replyBoardsByEntry(answered, [])
        .get("a1")
        ?.map((item) => [item.artifact.id, item.tool]),
    ).toEqual([["art_new", "show_answer"]]);

    const { place: _place, ...unasked } = SPEC;
    const shown = [
      user("u1"),
      reply("c1", "", [showBoard("call_art_new", unasked)]),
      reply("a1", link("art_new")),
    ];
    expect(replyBoardsByEntry(shown, []).size).toBe(0);
  });

  it("keeps the arguments null when a show_answer call is not in view", () => {
    const entries = [
      user("u1"),
      reply("c1", "", [showAnswer("call_art_a", {})]),
      reply("a1", link("art_a")),
    ];
    expect(replyBoardsByEntry(entries, [board("art_a")]).get("a1")?.[0].spec).toBeNull();
  });
});

describe("linkedArtifactIds", () => {
  it("reads the artifacts a reply links by id, and ignores a link that is not one", () => {
    expect([
      ...linkedArtifactIds("See [A](artifact:art_a), [B](artifact:art_b) and [C](https://x.test)."),
    ]).toEqual(["art_a", "art_b"]);
    expect([...linkedArtifactIds("[A](artifact:six-late-loads)")]).toEqual([]);
  });
});

function text(value: string): TurnSegment {
  return { kind: "text", text: value, closed: true };
}

function call(fields: Partial<Extract<TurnSegment, { kind: "tool" }>>): TurnSegment {
  return {
    kind: "tool",
    callId: "call_art_a",
    name: "show_board",
    arguments: SPEC,
    status: "running",
    content: "",
    ...fields,
  };
}

/**
 * A show_board result as the runtime saves and streams it (`runtimetools.BoardContent`
 * fenced by `toolresult.Fence`): a header line, the untrusted-data fence around the
 * JSON line, a blank line and the board's brief, then the note after the fence.
 */
function boardResult(place: "conversation" | "beside"): string {
  const json = `{"artifactId":"art_a","title":"Six late loads","status":"shown","place":"${place}"}`;
  const brief =
    "# Six late loads\n\nThere are 6 late loads.\n\n## The late loads\n- S-24117, 1h 40m behind";
  const note =
    place === "conversation"
      ? "[The board was drawn in the conversation as your reply, and the turn ended with it.]"
      : "[The board is beside the conversation, read live whenever it is opened.]";
  return `Result from show_board:\n<untrusted_data>\n${json}\n\n${brief}\n</untrusted_data>\n\n${note}`;
}

function announced(callId = "call_art_a"): AssistantArtifactEvent {
  return {
    id: "art_a",
    kind: "board",
    status: "Ready",
    title: "Six late loads",
    sourceToolCallId: callId,
    path: "",
    draft: null,
  };
}

const NOTHING = { artifacts: [], announced: [], savedCalls: new Set<string>() };

/*
While the turn runs, a show_board call still running is taken for a board in
the conversation unless it asked to be beside or holds too many blocks. A
finished call's place is the board's own once the thread lists it, and the
same guess while only the turn has announced it; its fenced result is never
read. The words written before the call stay; the reply the server writes for
the board after it never shows as words. Once the reply is saved, the saved
reply draws the board and the running turn draws nothing of it.
*/
describe("pendingReplyBoard", () => {
  it("takes a running call for a board in the conversation, with the words written before it as working", () => {
    expect(
      pendingReplyBoard(
        [text("Checking the lanes. "), call({}), text("There are 6 late loads.")],
        NOTHING,
      ),
    ).toEqual({
      tool: "show_board",
      spec: SPEC,
      kept: "",
      working: "Checking the lanes.",
      saved: false,
    });
  });

  it("does not take a running call that asked to be beside, or that holds more than four blocks", () => {
    expect(
      pendingReplyBoard([call({ arguments: { ...SPEC, place: "beside" } })], NOTHING),
    ).toBeNull();
    expect(
      pendingReplyBoard(
        [
          call({
            arguments: { ...SPEC, blocks: Array.from({ length: 5 }, () => ({ type: "records" })) },
          }),
        ],
        NOTHING,
      ),
    ).toBeNull();
  });

  it("keeps a finished call whose board the thread lists in the conversation, past the reply streamed after it", () => {
    const segments = [
      call({ status: "done", content: boardResult("conversation") }),
      text("There are 6 late loads ([Six late loads](artifact:art_a))."),
    ];

    expect(
      pendingReplyBoard(segments, {
        ...NOTHING,
        artifacts: [board("art_a")],
        announced: [announced()],
      }),
    ).toEqual({ tool: "show_board", spec: SPEC, kept: "", working: "", saved: false });
  });

  it("drops a finished call whose listed board is beside the conversation, whatever it asked", () => {
    const segments = [call({ status: "done", content: boardResult("beside") })];

    expect(
      pendingReplyBoard(segments, {
        ...NOTHING,
        artifacts: [board("art_a", { payload: { place: "beside" } })],
        announced: [announced()],
      }),
    ).toBeNull();
  });

  it("guesses as for a running call while only the turn has announced the board", () => {
    const done = (args: Record<string, unknown>) => [
      call({ status: "done", content: boardResult("conversation"), arguments: args }),
    ];
    const context = { ...NOTHING, announced: [announced()] };

    expect(pendingReplyBoard(done(SPEC), context)).toEqual({
      tool: "show_board",
      spec: SPEC,
      kept: "",
      working: "",
      saved: false,
    });
    expect(pendingReplyBoard(done({ ...SPEC, place: "beside" }), context)).toBeNull();
  });

  it("draws nothing for a finished call that made no board, nor for a failed one", () => {
    expect(
      pendingReplyBoard([call({ status: "done", content: boardResult("conversation") })], NOTHING),
    ).toBeNull();
    expect(
      pendingReplyBoard([call({ status: "failed", content: 'Tool "show_board" failed:' })], {
        ...NOTHING,
        announced: [announced()],
      }),
    ).toBeNull();
    expect(
      pendingReplyBoard([call({ status: "done" })], {
        ...NOTHING,
        announced: [announced("call_other")],
      }),
    ).toBeNull();
    expect(pendingReplyBoard([text("No board here.")], NOTHING)).toBeNull();
  });

  it("says the board is drawn by the saved reply once its call is saved, so nothing of it streams", () => {
    expect(
      pendingReplyBoard(
        [
          text("Checking. "),
          call({ status: "done", content: boardResult("conversation") }),
          text("There are 6."),
        ],
        {
          artifacts: [board("art_a")],
          announced: [announced()],
          savedCalls: new Set(["call_art_a"]),
        },
      ),
    ).toEqual({ tool: "show_board", spec: SPEC, kept: "", working: "", saved: true });
  });
});

/*
A board placed in the conversation is the reply and ends the turn, so the notes
the agent wrote between its tool calls before it were never the answer. Drawn
above the board they made one question read as a run of replies.
*/
describe("workingSteps", () => {
  const boardsFor = (entries: ThreadEntry[]) =>
    replyBoardsByEntry(entries, [board("art_a", { sourceToolCallId: "call_art_a" })]);

  it("is every earlier step of an answer that ends on a board", () => {
    const entries = [
      user("u1"),
      reply("s1", "I'll pull the late loads."),
      reply("s2", "The board shows 40 late moves.", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    expect([...workingSteps(entries, boardsFor(entries))]).toEqual(["s1", "s2"]);
  });

  it("leaves an answer in words whole: its steps are its paragraphs", () => {
    const entries = [user("u1"), reply("s1", "First I checked."), reply("a1", "Two are late.")];

    expect(workingSteps(entries, boardsFor(entries)).size).toBe(0);
  });

  it("keeps a step whose own card stands under it, and what came before that card", () => {
    const ask: ToolExchange = {
      call: { id: "call_ask", name: "ask_user", arguments: {} },
      result: null,
    };
    const entries = [
      user("u1"),
      reply("s1", "Before the question."),
      reply("q1", "Which customer?", [ask]),
      reply("s2", "Now I have it.", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    expect([...workingSteps(entries, boardsFor(entries))]).toEqual(["s2"]);
  });

  it("never reaches into another question's answer", () => {
    const entries = [
      user("u1"),
      reply("a0", "An earlier answer in words."),
      user("u2"),
      reply("s1", "Working.", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    expect([...workingSteps(entries, boardsFor(entries))]).toEqual(["s1"]);
  });
});

/** One model step of a turn: the words it wrote and the tools it asked for, in order. */
type ScriptStep = { text: string; tools: string[] };

/** The call a step makes to a tool, by the ids the runtime gives; the board's call is call_art_a. */
function scriptCallId(step: number, tool: string): string {
  return tool === "show_answer" ? "call_art_a" : `call_${step}_${tool}`;
}

/** The turn as it streams: each step's words as a finished message, then its calls. */
function streamed(script: readonly ScriptStep[]): TurnSegment[] {
  return script.flatMap((step, index) => [
    ...(step.text === "" ? [] : [text(step.text)]),
    ...step.tools.map((tool) =>
      call({
        callId: scriptCallId(index, tool),
        name: tool,
        arguments: tool === "show_answer" ? ANSWER : {},
        status: tool === "show_answer" ? "running" : "done",
      }),
    ),
  ]);
}

/** The same turn as saved: one reply per step, then the reply the server writes for the board. */
function saved(script: readonly ScriptStep[]): ThreadEntry[] {
  return [
    user("u1"),
    ...script.map((step, index) =>
      reply(
        `s${index}`,
        step.text.trim(),
        step.tools.map((tool) => ({
          call: {
            id: scriptCallId(index, tool),
            name: tool,
            arguments: tool === "show_answer" ? ANSWER : {},
          },
          result: null,
        })),
      ),
    ),
    reply("a1", link("art_a")),
  ];
}

/** The live split and the saved one, read through the same rule. */
function bothWays(script: readonly ScriptStep[]) {
  const live = pendingReplyBoard(streamed(script), NOTHING);
  const entries = saved(script);
  const working = workingSteps(
    entries,
    replyBoardsByEntry(entries, [board("art_a", { sourceToolCallId: "call_art_a" })]),
  );
  const words = (folded: boolean) =>
    script
      .filter((_, index) => working.has(`s${index}`) === folded)
      .map((step) => step.text)
      .join("")
      .trim();
  return { live, saved: { working: [...working], kept: words(false), notes: words(true) } };
}

/*
The running turn and the saved one decide which notes are only the working
before a board by one rule: once the board is known, the notes the saved turn
leaves out are the ones the running turn folds away, and the words it keeps are
the ones the saved turn draws. Before the board is known nothing folds.
*/
describe("one rule for the notes before a board, live and saved", () => {
  it("folds every note of an answer that ends on a board, live and saved alike", () => {
    const { live, saved: kept } = bothWays([
      { text: "I'll find the open GlobalTrade loads. ", tools: ["search_records"] },
      { text: "Found four of them. ", tools: ["show_answer"] },
    ]);

    expect(live).toMatchObject({
      kept: "",
      working: "I'll find the open GlobalTrade loads. Found four of them.",
    });
    expect(kept.working).toEqual(["s0", "s1"]);
    expect(live?.kept).toBe(kept.kept);
    expect(live?.working).toBe(kept.notes);
  });

  it("folds nothing while the turn has no board in the conversation", () => {
    expect(
      pendingReplyBoard(
        [
          text("I'll find the open loads. "),
          call({ callId: "call_1", name: "search_records", arguments: {}, status: "done" }),
        ],
        NOTHING,
      ),
    ).toBeNull();
  });

  it("keeps a step whose card stands under it, and what came before it, live and saved alike", () => {
    const { live, saved: kept } = bothWays([
      { text: "Looking up the lanes. ", tools: ["search_records"] },
      { text: "Running the lateness report. ", tools: ["report_run"] },
      { text: "Here is who they wait on. ", tools: ["show_answer"] },
    ]);

    expect(live).toMatchObject({
      kept: "Looking up the lanes. Running the lateness report.",
      working: "Here is who they wait on.",
    });
    expect(kept.working).toEqual(["s2"]);
    expect(live?.kept).toBe(kept.kept);
    expect(live?.working).toBe(kept.notes);
  });

  it("keeps a step that ran a report alongside the board, folding nothing", () => {
    const { live, saved: kept } = bothWays([
      { text: "Checking. ", tools: ["search_records"] },
      { text: "Running it and drawing it. ", tools: ["report_run", "show_answer"] },
    ]);

    expect(live).toMatchObject({ kept: "Checking. Running it and drawing it.", working: "" });
    expect(kept.working).toEqual([]);
  });

  it("counts a step that wrote no words, without words of its own to fold", () => {
    const { live, saved: kept } = bothWays([
      { text: "", tools: ["search_records"] },
      { text: "Found four. ", tools: ["show_answer"] },
    ]);

    expect(live).toMatchObject({ kept: "", working: "Found four." });
    expect(kept.working).toEqual(["s0", "s1"]);
  });

  it("folds only the question whose answer is a board, never the one before or after it", () => {
    const entries = [
      user("u1"),
      reply("s1", "I'll pull the late loads."),
      reply("s2", "", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
      user("u2"),
      reply("b1", "First I checked the dock."),
      reply("b2", "Two are still waiting."),
    ];

    expect([
      ...workingSteps(
        entries,
        replyBoardsByEntry(entries, [board("art_a", { sourceToolCallId: "call_art_a" })]),
      ),
    ]).toEqual(["s1", "s2"]);
  });

  it("does not split an answer where the person steered it or a record changed, as the running turn does not", () => {
    const entries: ThreadEntry[] = [
      user("u1"),
      reply("s1", "I'll pull the late loads."),
      { kind: "steer", message: { ...message("st1", "Only Dallas."), role: "User" } },
      { kind: "world", message: message("w1", "") },
      reply("s2", "Narrowed to Dallas.", [showBoard("call_art_a")]),
      reply("a1", link("art_a")),
    ];

    expect([
      ...workingSteps(
        entries,
        replyBoardsByEntry(entries, [board("art_a", { sourceToolCallId: "call_art_a" })]),
      ),
    ]).toEqual(["s1", "s2"]);
  });
});
