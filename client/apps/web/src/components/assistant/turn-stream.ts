import type {
  AssistantArtifactEvent,
  AssistantEntityRef,
  AssistantMessageAttachment,
  AssistantPageContext,
  AssistantStreamEvent,
  AssistantThread,
  SendMessageResult,
  RetryKind,
  ToolEffect,
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
  /** When this reader saw the call start and finish, in epoch milliseconds. */
  startedAt?: number;
  finishedAt?: number;
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

export type TurnState = {
  status: TurnStatus;
  userContent: string;
  /** The page the question was asked from, shown on the provisional user turn. */
  pageContext: AssistantPageContext | null;
  refusal: { message: string; reason: string; category: string } | null;
  segments: TurnSegment[];
  error: string | null;
  result: SendMessageResult | null;
  /** Set while the reply is starting over after a model died partway. */
  retrying: { attempt: number; provider: string; kind: RetryKind; waitSeconds: number } | null;
  /** What the turn has produced so far, as announced, so the pane can open it early. */
  artifacts: AssistantArtifactEvent[];
  /** The files and records the person handed over, shown on their provisional turn. */
  attachments: AssistantMessageAttachment[];
  mentions: AssistantEntityRef[];
  /** The turn follows up a decision; it carries no words of the person's own. */
  followUp: boolean;
  /** The thread a quick question was answered on, once the server names it. */
  thread: AssistantThread | null;
  /** When this reader began following the turn, in epoch milliseconds. */
  startedAt: number;
};

/** What a person hands over with a message besides the words. */
export type TurnContext = {
  attachments?: AssistantMessageAttachment[];
  mentions?: AssistantEntityRef[];
  /** The turn reports a decision rather than answering words of the person's own. */
  followUp?: boolean;
  /** When the turn began, in epoch milliseconds; now when not given. */
  startedAt?: number;
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
    result: null,
    retrying: null,
    artifacts: [],
    attachments: context.attachments ?? [],
    mentions: context.mentions ?? [],
    followUp: context.followUp ?? false,
    thread: null,
    startedAt: context.startedAt ?? Date.now(),
  };
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
      const last = state.segments.at(-1);
      if (last && last.kind === "reasoning" && !last.closed) {
        const segments = state.segments.slice(0, -1);
        segments.push({ ...last, text: last.text + event.data.text });
        return { ...state, status: "streaming", segments };
      }
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: [...state.segments, { kind: "reasoning", text: event.data.text, closed: false }],
      };
    }

    case "delta": {
      const segments = closeReasoning(state.segments);
      const last = segments.at(-1);
      if (last && last.kind === "text" && !last.closed) {
        const rest = segments.slice(0, -1);
        rest.push({ ...last, text: last.text + event.data.text });
        return { ...state, status: "streaming", retrying: null, segments: rest };
      }
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: [...segments, { kind: "text", text: event.data.text, closed: false }],
      };
    }

    case "message": {
      const segments = closeReasoning(state.segments);
      const last = segments.at(-1);
      if (last && last.kind === "text" && !last.closed) {
        segments[segments.length - 1] = {
          ...last,
          text: event.data.content !== "" ? event.data.content : last.text,
          closed: true,
        };
      } else if (event.data.content !== "") {
        segments.push({ kind: "text", text: event.data.content, closed: true });
      }
      return { ...state, status: "working", segments };
    }

    case "tool_started":
      return {
        ...state,
        status: "working",
        segments: [
          ...closeReasoning(state.segments),
          {
            kind: "tool",
            callId: event.data.callId,
            name: event.data.name,
            arguments: event.data.arguments ?? {},
            status: "running",
            content: "",
            effect: event.data.effect,
          },
        ],
      };

    case "tool_finished": {
      const status: ToolSegment["status"] = event.data.failed
        ? "failed"
        : event.data.proposed
          ? "proposed"
          : "done";
      const segments = state.segments.map((segment) =>
        segment.kind === "tool" && segment.callId === event.data.callId
          ? {
              ...segment,
              status,
              content: event.data.content,
              effect: event.data.effect ?? segment.effect,
              summary: event.data.summary || undefined,
            }
          : segment,
      );
      return { ...state, status: "working", segments };
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
        },
        segments: event.data.kind === "busy" ? state.segments : withdrawAttempt(state.segments),
      };

    case "artifact": {
      const known = state.artifacts.findIndex((artifact) => artifact.id === event.data.id);
      const artifacts =
        known === -1
          ? [...state.artifacts, event.data]
          : state.artifacts.map((artifact, index) => (index === known ? event.data : artifact));
      return { ...state, artifacts };
    }

    case "thread":
      return { ...state, thread: event.data };

    case "done":
      // A refusal is complete in itself; done after it only says the turn
      // was saved, and must not turn the refusal back into a reply.
      if (state.status === "refused") {
        return { ...state, result: event.data };
      }
      return { ...state, status: "done", result: event.data };

    case "error":
      return { ...state, status: "error", error: event.data.message };

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
  let changed = false;
  const segments = state.segments.map((segment) => {
    if (segment.kind !== "tool") {
      return segment;
    }
    const startedAt = segment.startedAt ?? now;
    const finishedAt =
      segment.status === "running" ? segment.finishedAt : (segment.finishedAt ?? now);
    if (startedAt === segment.startedAt && finishedAt === segment.finishedAt) {
      return segment;
    }
    changed = true;
    return { ...segment, startedAt, finishedAt };
  });

  return changed ? { ...state, segments } : state;
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
