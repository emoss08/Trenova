import type {
  AssistantArtifactEvent,
  AssistantEntityRef,
  AssistantMessageAttachment,
  AssistantPageContext,
  AssistantStreamEvent,
  AssistantThread,
  DelegateReport,
  FailedProvider,
  SendMessageResult,
  RetryKind,
  SavedMemory,
  StepRationale,
  ToolEffect,
  ToolVerdict,
  WatchedRecordChange,
} from "@/types/assistant";

/**
 * Where a turn is, from the reader's side of the wire.
 *
 * - guarding: sent, waiting to hear whether the question is in scope
 * - streaming: reply text, or the thinking before it, is arriving
 * - working: a tool is running, or the model is thinking between tools
 * - done / refused / error: the turn is over and the thread will be refetched
 */
export type TurnStatus = "guarding" | "streaming" | "working" | "done" | "refused" | "error";

export type TextSegment = {
  kind: "text";
  text: string;
  /** Closed text is a message the model finished before asking for a tool. */
  closed: boolean;
};

export type ToolSegment = {
  kind: "tool";
  callId: string;
  name: string;
  arguments: Record<string, unknown>;
  status: "running" | "done" | "failed" | "proposed";
  content: string;
  /** What the call does, as the server classifies it; absent from an older server. */
  effect?: ToolEffect;
  /** The server's one-line account of the result, once it has finished. */
  summary?: string;
  /** How the runtime judged the call once it finished; absent from an older server. */
  verdict?: ToolVerdict;
  /** When this reader saw the call start and finish, in epoch milliseconds. */
  startedAt?: number;
  finishedAt?: number;
  /** Set on a delegate_task call: the other agent's work on the task, nested under it. */
  delegate?: DelegateProgress;
  /** Why the agent took the step, when it said. */
  why?: StepRationale | null;
  /** The person's other agents a find_tools answer named as holding what this agent lacks. */
  handOffAgents?: readonly string[];
};

export type TurnRetry = {
  attempt: number;
  provider: string;
  kind: RetryKind;
  waitSeconds: number;
  /** How many times the model is asked in all; 0 when the server did not say. */
  maxAttempts: number;
};

/**
 * Another agent's work on a task the turn's agent handed it. Its words, its
 * thinking and its calls are kept here, under the call that handed the task
 * over, and never among the turn's own: a plain delta from it would read as
 * the reply being written.
 */
export type DelegateProgress = {
  agentId: string;
  agentName: string;
  icon: string;
  accent: string;
  task: string;
  segments: TurnSegment[];
  /** Set while its reply is starting over; its own, never the turn's. */
  retrying: TurnRetry | null;
  /** How the task ended and what it came to, once it has. */
  report: DelegateReport | null;
};

/**
 * What the model thought before the text that follows. A heavy model can sit
 * silent for a minute before its first word; this is what that minute shows.
 */
export type ReasoningSegment = {
  kind: "reasoning";
  text: string;
  /** Closed once the reply or a tool call begins; the thought is then finished. */
  closed: boolean;
};

export type TurnSegment = TextSegment | ToolSegment | ReasoningSegment;

/** Closes an open reasoning segment, if the last one is open. */
function closeReasoning(segments: TurnSegment[]): TurnSegment[] {
  const last = segments.at(-1);
  if (last && last.kind === "reasoning" && !last.closed) {
    return [...segments.slice(0, -1), { ...last, closed: true }];
  }
  return segments;
}

/**
 * Withdraws what the attempt being retried had streamed: the open reply and
 * the thinking that led into it. Everything up to the last finished message
 * or tool call belongs to model calls that completed, so it stays.
 */
function withdrawAttempt(segments: TurnSegment[]): TurnSegment[] {
  let end = segments.length;
  while (end > 0) {
    const segment = segments[end - 1];
    if (segment.kind === "tool" || (segment.kind === "text" && segment.closed)) {
      break;
    }
    end -= 1;
  }
  return end === segments.length ? segments : segments.slice(0, end);
}

/**
 * Puts the reply the turn recorded in place of the one that streamed, when the
 * runtime corrected it afterwards. Only the attempt's open reply goes: the
 * thinking that led into it happened and stays, as does everything the turn
 * finished before it.
 */
function replaceReply(segments: TurnSegment[], text: string): TurnSegment[] {
  const attempt = withdrawAttempt(segments).length;
  const kept =
    attempt === segments.length
      ? segments
      : [
          ...segments.slice(0, attempt),
          ...segments.slice(attempt).filter((segment) => segment.kind !== "text"),
        ];
  return appendDelta(kept, text);
}

/** Thinking arrives: it grows the open thought or opens a new one. */
function appendReasoning(segments: TurnSegment[], text: string): TurnSegment[] {
  const last = segments.at(-1);
  if (last && last.kind === "reasoning" && !last.closed) {
    return [...segments.slice(0, -1), { ...last, text: last.text + text }];
  }
  return [...segments, { kind: "reasoning", text, closed: false }];
}

/** Reply text arrives: the thought before it is finished, and the open text grows. */
function appendDelta(segments: TurnSegment[], text: string): TurnSegment[] {
  const closed = closeReasoning(segments);
  const last = closed.at(-1);
  if (last && last.kind === "text" && !last.closed) {
    return [...closed.slice(0, -1), { ...last, text: last.text + text }];
  }
  return [...closed, { kind: "text", text, closed: false }];
}

/** A message boundary: the open text is the model's finished message. */
function closeMessage(segments: TurnSegment[], content: string): TurnSegment[] {
  const closed = closeReasoning(segments);
  const last = closed.at(-1);
  if (last && last.kind === "text" && !last.closed) {
    return [
      ...closed.slice(0, -1),
      { ...last, text: content !== "" ? content : last.text, closed: true },
    ];
  }
  if (content !== "") {
    return [...closed, { kind: "text", text: content, closed: true }];
  }
  return closed;
}

type ToolStartedData = Extract<AssistantStreamEvent, { event: "tool_started" }>["data"];
type ToolFinishedData = Extract<AssistantStreamEvent, { event: "tool_finished" }>["data"];

/**
 * A call begins. A call already on the list — opened early by the hand-off
 * it carries, or announced twice to a reader who rejoined — is updated in
 * place rather than listed again.
 */
function startTool(segments: TurnSegment[], data: ToolStartedData): TurnSegment[] {
  const closed = closeReasoning(segments);
  const known = closed.findIndex(
    (segment) => segment.kind === "tool" && segment.callId === data.callId,
  );
  if (known !== -1) {
    return closed.map((segment, index) =>
      index === known && segment.kind === "tool"
        ? {
            ...segment,
            name: data.name,
            arguments: data.arguments ?? segment.arguments,
            effect: data.effect ?? segment.effect,
            why: data.why ?? segment.why,
          }
        : segment,
    );
  }
  return [
    ...closed,
    {
      kind: "tool",
      callId: data.callId,
      name: data.name,
      arguments: data.arguments ?? {},
      status: "running",
      content: "",
      effect: data.effect,
      why: data.why,
    },
  ];
}

function finishTool(segments: TurnSegment[], data: ToolFinishedData): TurnSegment[] {
  const status: ToolSegment["status"] = data.failed
    ? "failed"
    : data.proposed
      ? "proposed"
      : "done";
  return segments.map((segment) =>
    segment.kind === "tool" && segment.callId === data.callId
      ? {
          ...segment,
          status,
          content: data.content,
          effect: data.effect ?? segment.effect,
          summary: data.summary || undefined,
          verdict: data.verdict,
          handOffAgents:
            data.handOffAgents && data.handOffAgents.length > 0 ? data.handOffAgents : undefined,
        }
      : segment,
  );
}

/** No agent is named. One shared value, so a reader keyed on it does not run again. */
export const NO_HAND_OFF_AGENTS: readonly string[] = Object.freeze([]);

/**
 * The agents the turn's own newest find_tools answer named as holding what its
 * agent could not load. It is the step's own list, so the same answer is the
 * same value from one event to the next. Another agent's steps on a task it
 * was handed are nested under the call that handed it over, so they are not
 * read.
 */
export function turnHandOffAgents(
  segments: readonly TurnSegment[],
  interjections: readonly TurnInterjection[] = [],
): readonly string[] {
  let tools = segments.reduce((count, segment) => count + (segment.kind === "tool" ? 1 : 0), 0);
  for (let index = segments.length - 1; index >= 0; index -= 1) {
    const segment = segments[index];
    if (segment.kind !== "tool") {
      continue;
    }
    tools -= 1;
    if (segment.handOffAgents && segment.handOffAgents.length > 0) {
      const steeredAfter = interjections.some(
        (item) => item.kind === "steer" && item.toolsBefore > tools,
      );
      return steeredAfter ? NO_HAND_OFF_AGENTS : segment.handOffAgents;
    }
  }
  return NO_HAND_OFF_AGENTS;
}

/** What a running turn says about handing its conversation to another agent. */
export type TurnHandoff = {
  /** The turn answers words of the person's own; false for one following up a decision. */
  asked: boolean;
  /** What its own newest find_tools answer named; empty while none has. */
  agentIds: readonly string[];
};

/** The tool an agent hands a task to another agent with. */
export const DELEGATE_TOOL = "delegate_task";

/** A hand-off nothing has been heard of yet but the agent it went to. */
export function emptyDelegateProgress(agentId: string): DelegateProgress {
  return {
    agentId,
    agentName: "",
    icon: "",
    accent: "",
    task: "",
    segments: [],
    retrying: null,
    report: null,
  };
}

/**
 * Applies a change to the hand-off a delegate event belongs to. The event
 * names the delegate_task call; a reader who joined after the call was
 * announced has no step for it yet, so one is opened rather than the
 * delegate's work being dropped or shown as the turn's own.
 */
function updateDelegate(
  state: TurnState,
  delegateCallId: string,
  agentId: string,
  update: (delegate: DelegateProgress) => DelegateProgress,
): TurnState {
  const known = state.segments.some(
    (segment) => segment.kind === "tool" && segment.callId === delegateCallId,
  );
  const segments: TurnSegment[] = known
    ? state.segments.map((segment) =>
        segment.kind === "tool" && segment.callId === delegateCallId
          ? {
              ...segment,
              effect: segment.effect ?? "delegate",
              delegate: update(segment.delegate ?? emptyDelegateProgress(agentId)),
            }
          : segment,
      )
    : [
        ...closeReasoning(state.segments),
        {
          kind: "tool",
          callId: delegateCallId,
          name: DELEGATE_TOOL,
          arguments: agentId === "" ? {} : { agentId },
          status: "running",
          content: "",
          effect: "delegate",
          delegate: update(emptyDelegateProgress(agentId)),
        },
      ];

  return { ...state, status: "working", segments };
}

/** The delegate_task call an event of another agent's turn belongs under; empty for the turn's own. */
function delegateScope(data: { delegateCallId?: string }): string {
  return data.delegateCallId ?? "";
}

/**
 * A cap that stopped a question before it was asked: the agent's monthly
 * budget, its daily run count, or the person's own monthly allowance.
 */
export type TurnLimit = {
  kind: "monthly_budget" | "daily_runs" | "person_allowance";
  used: string;
  limit: string;
  /** When the cap lifts, in epoch seconds; 0 when the server did not say. */
  resetsAt: number;
};

/**
 * What reached the turn while it worked: the person's words, read at the next
 * step, and records that changed elsewhere, which the agent was told about.
 */
export type TurnInterjection =
  | {
      kind: "steer";
      id: string;
      text: string;
      mentions: AssistantEntityRef[];
      /**
       * How many tool steps the turn had run when the person wrote this. Tool
       * steps are never withdrawn, so it orders the steer against them: a
       * hand-off suggestion from a step before it is one the person has
       * already written past.
       */
      toolsBefore: number;
    }
  | { kind: "world"; key: string; changes: WatchedRecordChange[] };

/** The turn the conversation's queue started once this one was saved. */
export type NextTurn = { turnId: string; queuedId: string; input: string };

export type TurnState = {
  status: TurnStatus;
  userContent: string;
  /** The page the question was asked from, shown on the provisional user turn. */
  pageContext: AssistantPageContext | null;
  refusal: { message: string; reason: string; category: string } | null;
  segments: TurnSegment[];
  error: string | null;
  /** Set when the question was refused because a usage cap was reached. */
  limit: TurnLimit | null;
  /** The models asked, when every one of them failed. */
  failedProviders: FailedProvider[];
  /** The person stopped the reply themselves. */
  stopped: boolean;
  /** Seconds to wait before sending again, when the server said too many too fast. */
  rateLimited: number | null;
  result: SendMessageResult | null;
  /** Set while the reply is starting over after a model died partway. */
  retrying: TurnRetry | null;
  /** What the turn has produced so far, as announced, so the pane can open it early. */
  artifacts: AssistantArtifactEvent[];
  /** The memories the turn has used so far, in the order it used them. */
  usedMemoryIds: string[];
  /** What the turn kept through remember, or offered to keep. */
  savedMemories: SavedMemory[];
  /** The files and records the person handed over, shown on their provisional turn. */
  attachments: AssistantMessageAttachment[];
  mentions: AssistantEntityRef[];
  /** The turn follows up a decision; it carries no words of the person's own. */
  followUp: boolean;
  /** The thread a quick question was answered on, once the server names it. */
  thread: AssistantThread | null;
  /** When this reader began following the turn, in epoch milliseconds. */
  startedAt: number;
  /** What reached the turn while it worked, in the order it arrived. */
  interjections: TurnInterjection[];
  /** The reply the queue started after this one, once it is named. */
  next: NextTurn | null;
};

/** What a person hands over with a message besides the words. */
export type TurnContext = {
  attachments?: AssistantMessageAttachment[];
  mentions?: AssistantEntityRef[];
  /** The turn reports a decision rather than answering words of the person's own. */
  followUp?: boolean;
  /** When the turn began, in epoch milliseconds; now when not given. */
  startedAt?: number;
  /** The agent the person handed the message to, in place of the conversation's own. */
  directedAgentId?: string;
};

export function initialTurnState(
  userContent: string,
  pageContext: AssistantPageContext | null = null,
  context: TurnContext = {},
): TurnState {
  return {
    status: "guarding",
    userContent,
    pageContext,
    refusal: null,
    segments: [],
    error: null,
    limit: null,
    failedProviders: [],
    stopped: false,
    rateLimited: null,
    result: null,
    retrying: null,
    artifacts: [],
    usedMemoryIds: [],
    savedMemories: [],
    attachments: context.attachments ?? [],
    mentions: context.mentions ?? [],
    followUp: context.followUp ?? false,
    thread: null,
    startedAt: context.startedAt ?? Date.now(),
    interjections: [],
    next: null,
  };
}

/** The queued messages the turn has read so far, by id. */
export function steeredIds(
  interjections: readonly TurnInterjection[] | undefined,
): ReadonlySet<string> {
  const ids = new Set<string>();
  for (const interjection of interjections ?? []) {
    if (interjection.kind === "steer") {
      ids.add(interjection.id);
    }
  }
  return ids;
}

/**
 * Applies one server event. Pure, so the whole turn can be replayed in a test
 * and the rendering is only ever a function of this state.
 */
export function reduceTurn(state: TurnState, event: AssistantStreamEvent): TurnState {
  switch (event.event) {
    case "accepted":
      return { ...state, status: "working" };

    case "refused":
      // A refusal of the answer arrives after its text has streamed; the
      // reader must not be left holding words the guard withdrew.
      return {
        ...state,
        status: "refused",
        refusal: {
          message: event.data.message,
          reason: event.data.reason,
          category: event.data.category,
        },
        segments: [],
      };

    case "reasoning": {
      // Thinking that continues an open thought is the same attempt; only a
      // new thought says a retry has been answered.
      const last = state.segments.at(-1);
      const continuing = last?.kind === "reasoning" && !last.closed;
      return {
        ...state,
        status: "streaming",
        retrying: continuing ? state.retrying : null,
        segments: appendReasoning(state.segments, event.data.text),
      };
    }

    case "delta":
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: appendDelta(state.segments, event.data.text),
      };

    case "message": {
      const scope = delegateScope(event.data);
      if (scope !== "") {
        const content = event.data.content;
        return updateDelegate(state, scope, event.data.agentId ?? "", (delegate) => ({
          ...delegate,
          segments: closeMessage(delegate.segments, content),
        }));
      }
      return {
        ...state,
        status: "working",
        segments: closeMessage(state.segments, event.data.content),
      };
    }

    case "tool_started": {
      const scope = delegateScope(event.data);
      if (scope !== "") {
        const data = event.data;
        return updateDelegate(state, scope, data.agentId ?? "", (delegate) => ({
          ...delegate,
          segments: startTool(delegate.segments, data),
        }));
      }
      return { ...state, status: "working", segments: startTool(state.segments, event.data) };
    }

    case "tool_finished": {
      const scope = delegateScope(event.data);
      if (scope !== "") {
        const data = event.data;
        return updateDelegate(state, scope, data.agentId ?? "", (delegate) => ({
          ...delegate,
          segments: finishTool(delegate.segments, data),
        }));
      }
      return { ...state, status: "working", segments: finishTool(state.segments, event.data) };
    }

    case "delegate_started": {
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        agentId: data.agentId,
        agentName: data.agentName,
        icon: data.icon,
        accent: data.accent,
        task: data.task,
      }));
    }

    case "delegate_reasoning": {
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        retrying: null,
        segments: appendReasoning(delegate.segments, data.text),
      }));
    }

    case "delegate_delta": {
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        retrying: null,
        segments: appendDelta(delegate.segments, data.text),
      }));
    }

    case "delegate_retrying": {
      // The other agent's restart withdraws only what it had streamed; the
      // reply being shown, and the turn's own steps, are untouched.
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        retrying: {
          attempt: data.attempt,
          provider: data.provider,
          kind: data.kind,
          waitSeconds: data.waitSeconds,
          maxAttempts: data.maxAttempts ?? 0,
        },
        segments: data.kind === "busy" ? delegate.segments : withdrawAttempt(delegate.segments),
      }));
    }

    case "delegate_reply_replaced": {
      // The other agent's corrected reply replaces only what it had streamed.
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        retrying: null,
        segments: replaceReply(delegate.segments, data.text),
      }));
    }

    case "delegate_finished": {
      const data = event.data;
      return updateDelegate(state, data.delegateCallId, data.agentId, (delegate) => ({
        ...delegate,
        agentId: data.agentId !== "" ? data.agentId : delegate.agentId,
        agentName: data.agentName !== "" ? data.agentName : delegate.agentName,
        retrying: null,
        segments: closeReasoning(delegate.segments),
        report: data,
      }));
    }

    case "retrying":
      // A restart withdraws what the retried attempt streamed, so the reader
      // is not left holding half of one answer under another. The messages
      // finished before it and the tools that ran are kept, because they did
      // happen. A busy retry happened before any word arrived, so there is
      // nothing to withdraw.
      return {
        ...state,
        status: "working",
        retrying: {
          attempt: event.data.attempt,
          provider: event.data.provider,
          kind: event.data.kind,
          waitSeconds: event.data.waitSeconds,
          maxAttempts: event.data.maxAttempts ?? 0,
        },
        segments: event.data.kind === "busy" ? state.segments : withdrawAttempt(state.segments),
      };

    case "reply_replaced":
      // The runtime corrected the reply after it streamed: ids taken out, a
      // reprinted table pointed to, a code block left out. The model was not
      // asked again, so this is no retry: the streamed reply is swapped for
      // the one recorded, the way the reader would have seen it arrive.
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: replaceReply(state.segments, event.data.text),
      };

    case "artifact": {
      const known = state.artifacts.findIndex((artifact) => artifact.id === event.data.id);
      const artifacts =
        known === -1
          ? [...state.artifacts, event.data]
          : state.artifacts.map((artifact, index) => (index === known ? event.data : artifact));
      return { ...state, artifacts };
    }

    case "artifact_removed":
      return {
        ...state,
        artifacts: state.artifacts.filter((artifact) => artifact.id !== event.data.id),
      };

    case "memory_used":
      // Each event names every memory used so far, so the latest stands.
      return { ...state, usedMemoryIds: event.data.ids };

    case "memory_saved": {
      const saved = event.data;
      const known = state.savedMemories.some((memory) => memory.id === saved.id);
      return {
        ...state,
        savedMemories: known
          ? state.savedMemories.map((memory) => (memory.id === saved.id ? saved : memory))
          : [...state.savedMemories, saved],
      };
    }

    case "thread":
      return { ...state, thread: event.data };

    case "steered": {
      // A reader that rejoins is sent the turn from its start, so a steer it
      // already holds is not added twice.
      const steer = event.data;
      if (state.interjections.some((item) => item.kind === "steer" && item.id === steer.id)) {
        return state;
      }
      return {
        ...state,
        interjections: [
          ...state.interjections,
          {
            kind: "steer",
            id: steer.id,
            text: steer.content,
            mentions: steer.mentions,
            toolsBefore: state.segments.filter((segment) => segment.kind === "tool").length,
          },
        ],
      };
    }

    case "world_changed": {
      const changes = event.data.changes;
      if (changes.length === 0) {
        return state;
      }
      const key = changes.map((change) => `${change.recordId}:${change.at}`).join("|");
      if (state.interjections.some((item) => item.kind === "world" && item.key === key)) {
        return state;
      }
      return {
        ...state,
        interjections: [...state.interjections, { kind: "world", key, changes }],
      };
    }

    case "next_turn":
      return {
        ...state,
        next: {
          turnId: event.data.turnId,
          queuedId: event.data.queuedId,
          input: event.data.input,
        },
      };

    case "done":
      // A refusal is complete in itself; done after it only says the turn
      // was saved, and must not turn the refusal back into a reply.
      if (state.status === "refused") {
        return { ...state, result: event.data };
      }
      return { ...state, status: "done", result: event.data };

    case "error":
      return {
        ...state,
        status: "error",
        error: event.data.message,
        limit: event.data.limit ?? null,
        failedProviders: event.data.providers ?? [],
      };

    default:
      return state;
  }
}

/**
 * Applies one server event and stamps the tool calls it started or finished
 * with the reader's clock, so the turn in progress can say how long each step
 * took and how long the whole thing has run. Kept apart from `reduceTurn` so
 * the reducer stays a pure function of the events alone.
 */
export function advanceTurn(
  state: TurnState,
  event: AssistantStreamEvent,
  now: number = Date.now(),
): TurnState {
  return stampToolTimes(reduceTurn(state, event), now);
}

function stampToolTimes(state: TurnState, now: number): TurnState {
  const segments = stampSegments(state.segments, now);

  return segments === state.segments ? state : { ...state, segments };
}

/**
 * Stamps every call on the list, and every call of a hand-off nested under
 * one. Returns the same list when nothing needed a stamp, so a render keyed
 * on it does not run again for an event that changed no call.
 */
function stampSegments(segments: TurnSegment[], now: number): TurnSegment[] {
  let changed = false;
  const stamped = segments.map((segment) => {
    if (segment.kind !== "tool") {
      return segment;
    }
    const startedAt = segment.startedAt ?? now;
    const finishedAt =
      segment.status === "running" ? segment.finishedAt : (segment.finishedAt ?? now);
    const nested = segment.delegate ? stampSegments(segment.delegate.segments, now) : undefined;
    const nestedChanged = segment.delegate !== undefined && nested !== segment.delegate.segments;
    if (startedAt === segment.startedAt && finishedAt === segment.finishedAt && !nestedChanged) {
      return segment;
    }
    changed = true;
    return {
      ...segment,
      startedAt,
      finishedAt,
      ...(nestedChanged && segment.delegate && nested
        ? { delegate: { ...segment.delegate, segments: nested } }
        : {}),
    };
  });

  return changed ? stamped : segments;
}

/** Why a turn stopped, from the reader's side. */
export type TurnFailureCause = "stopped" | "failed" | "ended";

/**
 * What the reader lost, which is what the failure copy has to say. A turn
 * with nothing on screen lost nothing but the answer; one with words or a
 * lookup on screen was cut off, and what is shown is what had happened.
 */
export type TurnFailureKind =
  | "stopped-before-start"
  | "stopped"
  | "failed-before-start"
  | "cut-off";

export function describeTurnFailure(state: TurnState, cause: TurnFailureCause): TurnFailureKind {
  const underway = state.segments.length > 0;
  if (cause === "stopped") {
    return underway ? "stopped" : "stopped-before-start";
  }
  return underway ? "cut-off" : "failed-before-start";
}

/** Whether the turn is still in flight, for disabling the composer. */
export function isTurnActive(state: TurnState | null): boolean {
  return state !== null && ["guarding", "streaming", "working"].includes(state.status);
}
