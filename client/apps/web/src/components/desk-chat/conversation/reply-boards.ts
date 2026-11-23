import type { ReplyBoard } from "@/components/assistant/artifact-opener";
import type { ThreadEntry, ToolCallRecord } from "@/components/assistant/thread-view";
import type { TurnSegment } from "@/components/assistant/turn-stream";
import { artifactRefId } from "@/lib/artifact-ref";
import { boardPlaceOf, isBoardCallTool, type BoardCallTool } from "@/lib/board-place";
import type { AssistantArtifact, AssistantArtifactEvent } from "@/types/assistant";

/**
 * The most blocks a show_board board placed in the conversation holds; one
 * with more is put beside it. The server's `board.PlaceFor` decides; this
 * guesses the same way while the board itself is not in view yet.
 */
const CONVERSATION_BLOCKS = 4;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * A board-making call's arguments, kept for the reply to lay the board out
 * from while it is read, or null when the call is not in view: a show_board
 * call has its blocks or verdict, a show_answer call its answer or groups.
 */
function callSpec(
  tool: BoardCallTool,
  args: Record<string, unknown> | null | undefined,
): Record<string, unknown> | null {
  if (!isRecord(args)) return null;
  const shaped =
    tool === "show_answer"
      ? "answer" in args || "groups" in args
      : "blocks" in args || "verdict" in args;
  return shaped ? args : null;
}

/**
 * Where the server will place a call's board, worked out from what the call
 * asked as `board.PlaceFor` does: a show_answer board is always in the
 * conversation; a show_board board only when the call asked for the
 * conversation and holds at most four blocks, and beside otherwise. Used only
 * until the board itself says where it is.
 */
function guessesConversation(
  tool: BoardCallTool,
  args: Record<string, unknown> | null | undefined,
): boolean {
  if (tool === "show_answer") return true;
  if (!isRecord(args)) return false;
  const blocks = Array.isArray(args.blocks) ? args.blocks.length : 0;
  return args.place === "conversation" && blocks <= CONVERSATION_BLOCKS;
}

const LINK = /\]\((artifact:[^)\s]+)\)/gu;

/** The artifacts a reply's words link, by id. */
export function linkedArtifactIds(content: string): Set<string> {
  const ids = new Set<string>();
  for (const match of content.matchAll(LINK)) {
    const id = artifactRefId(match[1]);
    if (id !== null) ids.add(id);
  }
  return ids;
}

function lineageOf(artifact: AssistantArtifact): string {
  return artifact.lineageId !== "" ? artifact.lineageId : artifact.id;
}

type Turn = {
  calls: ToolCallRecord[];
  /** The calls whose result came back failed: they showed nothing. */
  failed: Set<string>;
  replies: { id: string; content: string }[];
  threadId: string;
};

/**
 * A board the turn's saved reply links before the thread's artifact list holds
 * it, standing in until the list catches up. It carries what the board is read
 * by (its id and thread) and what the call asked; the board's own read says
 * the rest, and the listed artifact takes its place under the same id.
 */
function unlistedBoard(id: string, threadId: string, call: ToolCallRecord): AssistantArtifact {
  const title = isRecord(call.arguments) ? call.arguments.title : undefined;
  return {
    id,
    threadId,
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "board",
    status: "Ready",
    title: typeof title === "string" ? title : "",
    payload: { place: "conversation" },
    sourceToolCallId: call.id,
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "",
    turn: "",
    createdAt: 0,
    updatedAt: 0,
  };
}

/**
 * Which saved replies draw a board in the conversation, and which boards each
 * draws. A reply draws, in the order its turn showed them, every board its
 * turn placed in the conversation that is still the newest version of its
 * lineage (a revised board's own reply draws the revision; the earlier reply
 * goes back to its words). The reply that draws them is the one of that turn
 * whose words link one of them: the reply the server wrote for the board,
 * which ends the turn. A board placed beside and a turn whose replies link none
 * of its boards draw nothing new. A board is made by a show_board or a
 * show_answer call. The reply is saved before the thread's artifact list
 * catches up with the board, so a turn with one such call the list does not
 * hold yet, whose board the conversation takes,
 * and whose replies link one artifact the list does not hold, draws that
 * artifact: the reply never shows as words for the moment between.
 */
export function replyBoardsByEntry(
  entries: readonly ThreadEntry[],
  artifacts: readonly AssistantArtifact[],
): Map<string, ReplyBoard[]> {
  const byCall = new Map<string, AssistantArtifact>();
  const newest = new Map<string, number>();
  for (const artifact of artifacts) {
    if (artifact.kind !== "board") continue;
    if (artifact.sourceToolCallId !== "") byCall.set(artifact.sourceToolCallId, artifact);
    const lineage = lineageOf(artifact);
    newest.set(lineage, Math.max(newest.get(lineage) ?? 0, artifact.lineageSeq));
  }

  const known = new Set(artifacts.map((artifact) => artifact.id));

  const out = new Map<string, ReplyBoard[]>();
  const settle = (turn: Turn) => {
    // A call whose board the thread does not list yet: the reply the server
    // wrote for it is saved before the artifact list catches up.
    const unlisted = turn.calls.filter(
      (call) => isBoardCallTool(call.name) && !byCall.has(call.id) && !turn.failed.has(call.id),
    );
    const linked = turn.replies.flatMap((reply) =>
      [...linkedArtifactIds(reply.content)].filter((id) => !known.has(id)),
    );
    const lone = unlisted.length === 1 ? unlisted[0] : null;
    const provisional =
      lone &&
      isBoardCallTool(lone.name) &&
      guessesConversation(lone.name, lone.arguments) &&
      linked.length === 1
        ? { call: lone, id: linked[0] }
        : null;

    const boards: ReplyBoard[] = [];
    for (const call of turn.calls) {
      const tool = call.name;
      if (!isBoardCallTool(tool)) continue;
      const spec = callSpec(tool, call.arguments);
      if (provisional?.call === call) {
        boards.push({ artifact: unlistedBoard(provisional.id, turn.threadId, call), tool, spec });
        continue;
      }
      const artifact = byCall.get(call.id);
      if (!artifact || boardPlaceOf(artifact.payload) !== "conversation") continue;
      if (artifact.lineageSeq < (newest.get(lineageOf(artifact)) ?? 0)) continue;
      boards.push({ artifact, tool, spec });
    }
    if (boards.length === 0) return;
    const ids = new Set(boards.map((board) => board.artifact.id));
    for (let index = turn.replies.length - 1; index >= 0; index -= 1) {
      const reply = turn.replies[index];
      if ([...linkedArtifactIds(reply.content)].some((id) => ids.has(id))) {
        out.set(reply.id, boards);
        return;
      }
    }
  };

  let turn: Turn = { calls: [], failed: new Set(), replies: [], threadId: "" };
  for (const entry of entries) {
    if (entry.kind === "user") {
      settle(turn);
      turn = { calls: [], failed: new Set(), replies: [], threadId: "" };
    } else if (entry.kind === "assistant") {
      for (const tool of entry.tools) {
        turn.calls.push(tool.call);
        if (tool.result?.toolFailed) turn.failed.add(tool.call.id);
      }
      turn.replies.push({ id: entry.message.id, content: entry.message.content });
      turn.threadId = entry.message.threadId;
    }
  }
  settle(turn);
  return out;
}

/** The calls whose own cards a step draws under its words, which keep the step on screen. */
const CARD_TOOLS = new Set(["ask_user", "report_run"]);

/**
 * One model step as the fold reads it, saved or live: whether a card stands
 * under it, and whether it draws a board in the conversation. A break is where
 * one turn ends and the next begins.
 */
type FoldStep = { kind: "step"; id: string; card: boolean; board: boolean } | { kind: "break" };

/**
 * The one rule for which steps are only the agent's working: in each stretch
 * of a turn (from its start, or from a step whose card stands under it), every
 * step before the last one that draws a board. A board placed in the
 * conversation is the reply and ends the turn, so what came before it was never
 * the answer. A step whose card stands under it keeps its place and starts a
 * new stretch. A stretch that draws no board is an answer in words, left whole.
 * The saved entries and the running turn both ask this, so a note folded while
 * the turn runs is the note the saved turn leaves out.
 */
function foldedSteps(steps: Iterable<FoldStep>): Set<string> {
  const folded = new Set<string>();
  let stretch: { id: string; board: boolean }[] = [];
  const close = () => {
    let at = stretch.length - 1;
    while (at >= 0 && !stretch[at].board) at -= 1;
    for (const step of stretch.slice(0, Math.max(at, 0))) folded.add(step.id);
    stretch = [];
  };
  for (const step of steps) {
    if (step.kind === "break" || step.card) {
      close();
      continue;
    }
    stretch.push({ id: step.id, board: step.board });
  }
  close();
  return folded;
}

/** Entries that sit inside a turn without ending it: a decision, a steer, a change elsewhere. */
const WITHIN_TURN = new Set<ThreadEntry["kind"]>(["decision", "steer", "world"]);

function* savedFoldSteps(
  entries: readonly ThreadEntry[],
  boards: ReadonlyMap<string, readonly ReplyBoard[]>,
): Generator<FoldStep> {
  for (const entry of entries) {
    if (WITHIN_TURN.has(entry.kind)) continue;
    if (entry.kind !== "assistant") {
      yield { kind: "break" };
      continue;
    }
    yield {
      kind: "step",
      id: entry.message.id,
      card: entry.tools.some((tool) => CARD_TOOLS.has(tool.call.name)),
      board: (boards.get(entry.message.id)?.length ?? 0) > 0,
    };
  }
}

/**
 * The saved steps of an answer that are only the agent's working, once that
 * answer is a board drawn in the conversation (see `foldedSteps`): drawn above
 * the board, one question read as a run of replies. What reached the turn while
 * it worked (the person's steer, a record changed elsewhere) does not split
 * its answer, as it does not while the turn runs.
 */
export function workingSteps(
  entries: readonly ThreadEntry[],
  boards: ReadonlyMap<string, readonly ReplyBoard[]>,
): Set<string> {
  return foldedSteps(savedFoldSteps(entries, boards));
}

/** One model step of the running turn: the words it wrote and whether a card stands under it. */
type LiveStep = { text: string; card: boolean };

/**
 * The running turn's model steps up to the segment at `end`: a step is the
 * words a model call wrote and the tools it asked for, so a new one starts at
 * words after a tool, or after a message the model finished.
 */
function liveSteps(segments: readonly TurnSegment[], end: number): LiveStep[] {
  const steps: LiveStep[] = [];
  let current: LiveStep | null = null;
  let ended = false;
  for (const segment of segments.slice(0, end + 1)) {
    if (segment.kind === "tool") {
      if (current === null) {
        current = { text: "", card: false };
        steps.push(current);
      }
      if (CARD_TOOLS.has(segment.name)) current.card = true;
      ended = true;
      continue;
    }
    if (segment.kind !== "text") continue;
    if (current === null || ended) {
      current = { text: "", card: false };
      steps.push(current);
      ended = false;
    }
    current.text += segment.text;
    if (segment.closed) ended = true;
  }
  return steps;
}

/**
 * The running turn's words before the call placing its board in the
 * conversation, as the fold splits them: what stays (a stretch closed by a
 * step whose card stands under it) and what is only the working before the
 * board, which leaves the moment the board is known. The board stands as one
 * more step at the end, as the reply the server writes for it does once saved.
 */
function liveFold(segments: readonly TurnSegment[], at: number): { kept: string; working: string } {
  const steps = liveSteps(segments, at);
  const folded = foldedSteps([
    ...steps.map((step, index): FoldStep => ({
      kind: "step",
      id: String(index),
      card: step.card,
      board: false,
    })),
    { kind: "step", id: "board", card: false, board: true },
  ]);
  const join = (keep: boolean) =>
    steps
      .filter((_, index) => folded.has(String(index)) !== keep)
      .map((step) => step.text)
      .join("")
      .trim();
  return { kept: join(true), working: join(false) };
}

/** What tells the running turn's board apart from one that is not in the conversation. */
export type PendingBoardContext = {
  /** The thread's artifacts as listed. */
  artifacts: readonly AssistantArtifact[];
  /** What the running turn has announced it made so far. */
  announced: readonly AssistantArtifactEvent[];
  /** The calls whose replies are already saved in the thread. */
  savedCalls: ReadonlySet<string>;
};

/**
 * A board the running turn is placing in the conversation, before its reply
 * is saved: the tool and arguments of the call making it (show_board or
 * show_answer), and the words the turn wrote before that call, split as the
 * saved turn will be (`foldedSteps`): `kept`, drawn as the reply's words, and
 * `working`, the notes before the board, which leave as soon as it is known.
 * What follows the call is the reply the server writes for the board, which
 * the saved reply draws as the board.
 *
 * A running show_answer call is always taken for one in the conversation; a
 * running show_board call only when it asked for the conversation and holds at
 * most four blocks. A finished call
 * made a board only when the turn announced one for it or the thread lists
 * one; its place is the listed board's, or the same guess until the list holds
 * it. Its result is never read: the runtime fences every tool result as
 * untrusted data, so it is not the tool's JSON. A failed call and a finished
 * one that made no board draw nothing here. A call whose reply is already saved
 * in the thread comes back `saved`: the saved reply draws the board, so the
 * running turn draws neither the skeleton nor any of its words.
 */
export function pendingReplyBoard(
  segments: readonly TurnSegment[],
  context: PendingBoardContext,
): {
  tool: BoardCallTool;
  spec: Record<string, unknown> | null;
  kept: string;
  working: string;
  saved: boolean;
} | null {
  let at = -1;
  segments.forEach((segment, index) => {
    if (segment.kind === "tool" && isBoardCallTool(segment.name)) at = index;
  });
  const call = at >= 0 ? segments[at] : undefined;
  if (!call || call.kind !== "tool" || !isBoardCallTool(call.name)) return null;
  if (call.status !== "running" && call.status !== "done") return null;
  const tool = call.name;
  const spec = callSpec(tool, call.arguments);
  if (context.savedCalls.has(call.callId)) {
    return { tool, spec, kept: "", working: "", saved: true };
  }
  if (call.status === "running") {
    if (!guessesConversation(tool, call.arguments)) return null;
  } else {
    const listed = context.artifacts.find(
      (artifact) => artifact.kind === "board" && artifact.sourceToolCallId === call.callId,
    );
    if (listed) {
      if (boardPlaceOf(listed.payload) !== "conversation") return null;
    } else {
      const made = context.announced.some(
        (artifact) => artifact.kind === "board" && artifact.sourceToolCallId === call.callId,
      );
      if (!made || !guessesConversation(tool, call.arguments)) return null;
    }
  }
  return { tool, spec, ...liveFold(segments, at), saved: false };
}
